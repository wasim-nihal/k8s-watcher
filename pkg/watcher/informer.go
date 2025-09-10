package watcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/wasim-nihal/k8s-watcher/pkg/config"
	"github.com/wasim-nihal/k8s-watcher/pkg/logger"
)

// ResourceInformer manages the Kubernetes informer setup and event handling
type ResourceInformer struct {
	client     kubernetes.Interface
	config     *config.ResourceConfig
	handler    ResourceHandler
	namespaces []string
	stopCh     chan struct{}
}

// NewResourceInformer creates a new resource informer
func NewResourceInformer(client kubernetes.Interface, cfg *config.ResourceConfig, handler ResourceHandler) *ResourceInformer {
	var namespaces []string
	if cfg.ResourceNames != nil && len(cfg.ResourceNames) > 0 {
		// Extract namespaces from resource names
		nsMap := make(map[string]bool)
		for _, name := range cfg.ResourceNames {
			parts := strings.Split(name, "/")
			if len(parts) > 1 {
				nsMap[parts[0]] = true
			}
		}
		for ns := range nsMap {
			namespaces = append(namespaces, ns)
		}
	}

	return &ResourceInformer{
		client:     client,
		config:     cfg,
		handler:    handler,
		namespaces: namespaces,
		stopCh:     make(chan struct{}),
	}
}

// Start initializes and starts the informers
func (r *ResourceInformer) Start(ctx context.Context) error {
	if r.config.Method == config.WatchMethodList {
		return r.handleListMode(ctx)
	}

	// Set up informer factories for each namespace
	for _, ns := range r.getNamespaces() {
		factory := informers.NewSharedInformerFactoryWithOptions(
			r.client,
			time.Duration(r.config.WatchConfig.ServerTimeout)*time.Second,
			informers.WithNamespace(ns),
		)

		switch r.config.Type {
		case config.ResourceTypeConfigMap, config.ResourceTypeBoth:
			informer := factory.Core().V1().ConfigMaps().Informer()
			r.setupEventHandlers(informer, "ConfigMap")
		}

		if r.config.Type == config.ResourceTypeSecret || r.config.Type == config.ResourceTypeBoth {
			informer := factory.Core().V1().Secrets().Informer()
			r.setupEventHandlers(informer, "Secret")
		}

		factory.Start(r.stopCh)
	}

	// Wait for context cancellation
	<-ctx.Done()
	close(r.stopCh)
	return nil
}

// handleListMode handles the LIST method of operation
func (r *ResourceInformer) handleListMode(ctx context.Context) error {
	for _, ns := range r.getNamespaces() {
		if err := r.listResources(ctx, ns); err != nil {
			return err
		}
	}
	return nil
}

// listResources lists all matching resources in a namespace
func (r *ResourceInformer) listResources(ctx context.Context, namespace string) error {
	opts := metav1.ListOptions{}

	if r.config.Type == config.ResourceTypeConfigMap || r.config.Type == config.ResourceTypeBoth {
		cms, err := r.client.CoreV1().ConfigMaps(namespace).List(ctx, opts)
		if err != nil {
			return fmt.Errorf("listing configmaps in namespace %s: %w", namespace, err)
		}
		for _, cm := range cms.Items {
			r.handler.OnAdd(&cm)
		}
	}

	if r.config.Type == config.ResourceTypeSecret || r.config.Type == config.ResourceTypeBoth {
		secrets, err := r.client.CoreV1().Secrets(namespace).List(ctx, opts)
		if err != nil {
			return fmt.Errorf("listing secrets in namespace %s: %w", namespace, err)
		}
		for _, secret := range secrets.Items {
			r.handler.OnAdd(&secret)
		}
	}

	return nil
}

// setupEventHandlers configures the event handlers for the informer
func (r *ResourceInformer) setupEventHandlers(informer cache.SharedIndexInformer, resourceType string) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			r.handler.OnAdd(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			r.handler.OnUpdate(oldObj, newObj)
		},
		DeleteFunc: func(obj interface{}) {
			r.handler.OnDelete(obj)
		},
	})

	logger.Info("Set up event handlers",
		"resourceType", resourceType,
	)
}

// getNamespaces returns the list of namespaces to watch
func (r *ResourceInformer) getNamespaces() []string {
	if len(r.namespaces) > 0 {
		return r.namespaces
	}
	return []string{metav1.NamespaceAll}
}

// Stop stops the informer
func (r *ResourceInformer) Stop() {
	close(r.stopCh)
}
