package watcher

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/wasim-nihal/k8s-watcher/pkg/config"
	"github.com/wasim-nihal/k8s-watcher/pkg/file"
	"github.com/wasim-nihal/k8s-watcher/pkg/http"
	"github.com/wasim-nihal/k8s-watcher/pkg/label"
	"github.com/wasim-nihal/k8s-watcher/pkg/logger"
)

// ResourceHandler interface defines methods for handling resource events
type ResourceHandler interface {
	OnAdd(obj interface{})
	OnUpdate(oldObj, newObj interface{})
	OnDelete(obj interface{})
}

// Watcher implements the main watcher functionality
type Watcher struct {
	client            kubernetes.Interface
	config            *config.Config
	labelManager      *label.Manager
	fileHandler       *file.Handler
	informer          *ResourceInformer
	processedVersions map[string]string // tracks resourceVersion of processed resources
}

// NewWatcher creates a new watcher instance
func NewWatcher(client kubernetes.Interface, cfg *config.Config) *Watcher {
	w := &Watcher{
		client:            client,
		config:            cfg,
		labelManager:      label.NewManager(cfg.Resources.Labels),
		fileHandler:       file.NewHandler(cfg.Output),
		processedVersions: make(map[string]string),
	}

	w.informer = NewResourceInformer(client, cfg.Kubernetes.Namespace, &cfg.Resources, w)
	return w
}

// Start begins watching resources
func (w *Watcher) Start(ctx context.Context) error {
	logger.Info("Starting watcher",
		"resourceType", w.config.Resources.Type,
		"method", w.config.Resources.Method,
	)

	return w.informer.Start(ctx)
}

// OnAdd handles resource addition events
func (w *Watcher) OnAdd(obj interface{}) {
	w.handleResource("Added", obj)
}

// OnUpdate handles resource update events
func (w *Watcher) OnUpdate(oldObj, newObj interface{}) {
	w.handleResource("Updated", newObj)
}

// OnDelete handles resource deletion events
func (w *Watcher) OnDelete(obj interface{}) {
	w.handleResource("Deleted", obj)
}

// getResourceKey generates a unique key for a resource
func (w *Watcher) getResourceKey(metadata *metav1.ObjectMeta) string {
	return fmt.Sprintf("%s/%s/%s", metadata.Namespace, metadata.Name, metadata.ResourceVersion)
}

// isProcessed checks if a resource version has already been processed
func (w *Watcher) isProcessed(metadata *metav1.ObjectMeta) bool {
	if !w.config.Resources.WatchConfig.IgnoreProcessed {
		return false
	}

	key := fmt.Sprintf("%s/%s", metadata.Namespace, metadata.Name)
	lastVersion, exists := w.processedVersions[key]
	return exists && lastVersion == metadata.ResourceVersion
}

// markProcessed marks a resource version as processed
func (w *Watcher) markProcessed(metadata *metav1.ObjectMeta) {
	if !w.config.Resources.WatchConfig.IgnoreProcessed {
		return
	}

	key := fmt.Sprintf("%s/%s", metadata.Namespace, metadata.Name)
	w.processedVersions[key] = metadata.ResourceVersion
}

// handleResource processes a resource event
func (w *Watcher) handleResource(action string, obj interface{}) {
	var metadata *metav1.ObjectMeta
	var data map[string][]byte

	switch v := obj.(type) {
	case *corev1.ConfigMap:
		metadata = &v.ObjectMeta
		data = make(map[string][]byte)
		for k, v := range v.Data {
			data[k] = []byte(v)
		}
		for k, v := range v.BinaryData {
			data[k] = v
		}
	case *corev1.Secret:
		metadata = &v.ObjectMeta
		data = v.Data
	default:
		logger.Error("Unknown resource type", "type", fmt.Sprintf("%T", obj))
		return
	}

	// Check if we've already processed this version
	if w.isProcessed(metadata) {
		logger.Debug("Skipping already processed resource version",
			"name", metadata.Name,
			"namespace", metadata.Namespace,
			"resourceVersion", metadata.ResourceVersion,
		)
		return
	}

	// Check if the resource matches our label selectors
	matchingConfigs := w.labelManager.MatchLabels(metadata.Labels)
	if len(matchingConfigs) == 0 {
		return
	}

	logger.Info("Processing resource",
		"action", action,
		"name", metadata.Name,
		"namespace", metadata.Namespace,
		"matches", len(matchingConfigs),
	)

	// Process the resource for each matching configuration
	for _, cfg := range matchingConfigs {
		if err := w.processResource(metadata, data, cfg); err != nil {
			logger.Error("Failed to process resource",
				"name", metadata.Name,
				"namespace", metadata.Namespace,
				"error", err,
			)
		}
	}

	// Mark the resource as processed after successful processing
	w.markProcessed(metadata)
}

// processResource handles a single resource for a specific label configuration
func (w *Watcher) processResource(metadata *metav1.ObjectMeta, data map[string][]byte, cfg config.LabelConfig) error {
	// Write files
	basePath := w.fileHandler.GetAnnotationPath(metadata.Annotations)
	for key, content := range data {
		filePath := w.fileHandler.GetOutputPath(metadata.Name, metadata.Namespace, key)
		// Use basePath if it's different from the default
		if basePath != w.fileHandler.GetDefaultPath() {
			filePath = filepath.Join(basePath, key)
		}
		if err := w.fileHandler.WriteFile(filePath, content); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}
	}

	// Execute script if configured
	if cfg.Script.Path != "" {
		if err := w.executeScript(cfg.Script); err != nil {
			return fmt.Errorf("executing script: %w", err)
		}
	}

	// Send notification if configured
	if cfg.Request.URL != "" {
		client := http.NewClient(cfg.Request)
		payload := map[string]interface{}{
			"resource":  metadata.Name,
			"namespace": metadata.Namespace,
			"timestamp": time.Now().UTC(),
		}
		if err := client.SendNotification(payload); err != nil {
			return fmt.Errorf("sending notification: %w", err)
		}
	}

	return nil
}

// executeScript runs the configured script
func (w *Watcher) executeScript(cfg config.ScriptConfig) error {
	ctx := context.Background()
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(cfg.Timeout)*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", cfg.Path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("script execution failed: %w, output: %s", err, string(output))
	}

	logger.Info("Script executed successfully",
		"path", cfg.Path,
		"output", string(output),
	)

	return nil
}
