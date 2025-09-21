package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/wasim-nihal/k8s-watcher/pkg/config"
	"github.com/wasim-nihal/k8s-watcher/pkg/logger"
)

func init() {
	// Initialize logger for tests
	err := logger.Initialize(config.LoggingConfig{
		Level:  "INFO",
		Format: "LOGFMT",
	})
	if err != nil {
		panic("Failed to initialize logger for tests: " + err.Error())
	}
}

func TestNewWatcher(t *testing.T) {
	cfg := &config.Config{
		Resources: config.ResourceConfig{
			Type: config.ResourceTypeConfigMap,
			Labels: []config.LabelConfig{
				{Name: "app", Value: "test"},
			},
		},
	}
	client := fake.NewSimpleClientset()

	w := NewWatcher(client, cfg)
	assert.NotNil(t, w)
	assert.NotNil(t, w.labelManager)
	assert.NotNil(t, w.fileHandler)
	assert.NotNil(t, w.processedVersions)
}

func TestWatcher_HandleResource(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "watcher-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create namespace directory structure
	nsDir := filepath.Join(tempDir, "default")
	err = os.MkdirAll(nsDir, 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Output: config.OutputConfig{
			Folder:          tempDir,
			DefaultFileMode: "0644",
		},
		Resources: config.ResourceConfig{
			Type: config.ResourceTypeConfigMap,
			Labels: []config.LabelConfig{
				{
					Name:  "app",
					Value: "test",
				},
			},
			WatchConfig: config.WatchConfig{
				IgnoreProcessed: true,
			},
		},
	}

	client := fake.NewSimpleClientset()
	w := NewWatcher(client, cfg)

	// Test ConfigMap handling
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
			ResourceVersion: "1",
		},
		Data: map[string]string{
			"test.txt": "test content",
		},
	}

	// Test OnAdd
	w.OnAdd(cm)

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tempDir, "default", "test-cm", "test.txt"))
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))

	// Verify resource version is tracked
	key := "default/test-cm"
	version, exists := w.processedVersions[key]
	assert.True(t, exists, "Resource version should be tracked")
	assert.Equal(t, "1", version, "Resource version should match")

	// Test duplicate handling
	w.OnAdd(cm)
	assert.Equal(t, "1", w.processedVersions[key], "Resource version should remain unchanged")

	// Test Secret handling
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
			ResourceVersion: "1",
		},
		Data: map[string][]byte{
			"secret.txt": []byte("secret content"),
		},
	}

	w.OnAdd(secret)

	// Verify secret file was written
	content, err = os.ReadFile(filepath.Join(tempDir, "default", "test-secret", "secret.txt"))
	require.NoError(t, err)
	assert.Equal(t, "secret content", string(content))
}

func TestWatcher_ExecuteScript(t *testing.T) {
	cfg := &config.Config{
		Resources: config.ResourceConfig{
			Type: config.ResourceTypeConfigMap,
			Labels: []config.LabelConfig{
				{
					Name:  "app",
					Value: "test",
					Script: config.ScriptConfig{
						Path:    "echo 'test'",
						Timeout: 5,
					},
				},
			},
		},
	}

	client := fake.NewSimpleClientset()
	w := NewWatcher(client, cfg)

	err := w.executeScript(cfg.Resources.Labels[0].Script)
	assert.NoError(t, err)

	// Test script timeout
	cfg.Resources.Labels[0].Script.Path = "sleep 10"
	cfg.Resources.Labels[0].Script.Timeout = 1
	err = w.executeScript(cfg.Resources.Labels[0].Script)
	assert.Error(t, err)
}

func TestWatcher_Start(t *testing.T) {
	cfg := &config.Config{
		Resources: config.ResourceConfig{
			Type: config.ResourceTypeConfigMap,
			Labels: []config.LabelConfig{
				{Name: "app", Value: "test"},
			},
		},
	}

	client := fake.NewSimpleClientset()
	w := NewWatcher(client, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := w.Start(ctx)
	assert.NoError(t, err)
}
