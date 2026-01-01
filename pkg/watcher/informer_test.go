package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/wasim-nihal/k8s-watcher/pkg/config"
)

// MockResourceHandler is a mock implementation of ResourceHandler
type MockResourceHandler struct {
	mock.Mock
}

func (m *MockResourceHandler) OnAdd(obj interface{}) {
	m.Called(obj)
}

func (m *MockResourceHandler) OnUpdate(oldObj, newObj interface{}) {
	m.Called(oldObj, newObj)
}

func (m *MockResourceHandler) OnDelete(obj interface{}) {
	m.Called(obj)
}

func TestNewResourceInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	cfg := &config.ResourceConfig{
		Type:          config.ResourceTypeConfigMap,
		ResourceNames: []string{"default/test-cm"},
	}
	handler := &MockResourceHandler{}

	informer := NewResourceInformer(client, "default", cfg, handler)
	assert.NotNil(t, informer)
	assert.Equal(t, []string{"default"}, informer.namespaces)
}

func TestResourceInformer_ListMode(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := &MockResourceHandler{}

	// Create test ConfigMap
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{
			"test.txt": "test content",
		},
	}
	_, err := client.CoreV1().ConfigMaps("default").Create(context.Background(), cm, metav1.CreateOptions{})
	assert.NoError(t, err)

	cfg := &config.ResourceConfig{
		Type:   config.ResourceTypeConfigMap,
		Method: config.WatchMethodList,
	}

	handler.On("OnAdd", mock.Anything).Return()

	informer := NewResourceInformer(client, "default", cfg, handler)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = informer.Start(ctx)
	assert.NoError(t, err)

	handler.AssertCalled(t, "OnAdd", mock.Anything)
}

func TestResourceInformer_WatchMode(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := &MockResourceHandler{}

	cfg := &config.ResourceConfig{
		Type:   config.ResourceTypeConfigMap,
		Method: config.WatchMethodWatch,
		WatchConfig: config.WatchConfig{
			ServerTimeout: 30,
		},
	}

	handler.On("OnAdd", mock.Anything).Return()
	handler.On("OnUpdate", mock.Anything, mock.Anything).Return()
	handler.On("OnDelete", mock.Anything).Return()

	informer := NewResourceInformer(client, "default", cfg, handler)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start informer in background
	go func() {
		err := informer.Start(ctx)
		assert.NoError(t, err)
	}()

	// Wait for informer to start
	time.Sleep(50 * time.Millisecond)

	// Create test ConfigMap
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{
			"test.txt": "test content",
		},
	}

	_, err := client.CoreV1().ConfigMaps("default").Create(context.Background(), cm, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Wait for events to be processed
	time.Sleep(50 * time.Millisecond)

	handler.AssertNumberOfCalls(t, "OnAdd", 1)
}

func TestResourceInformer_Stop(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := &MockResourceHandler{}

	cfg := &config.ResourceConfig{
		Type: config.ResourceTypeConfigMap,
	}

	informer := NewResourceInformer(client, "default", cfg, handler)
	informer.Stop()

	// Verify the stop channel is closed
	select {
	case _, ok := <-informer.stopCh:
		assert.False(t, ok, "Stop channel should be closed")
	default:
		t.Error("Stop channel should be closed")
	}
}

func TestResourceInformer_GetNamespaces(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := &MockResourceHandler{}

	// Test with specific namespaces
	cfg := &config.ResourceConfig{
		Type:          config.ResourceTypeConfigMap,
		ResourceNames: []string{"ns1/cm1", "ns2/cm2"},
	}

	informer := NewResourceInformer(client, "default", cfg, handler)
	namespaces := informer.getNamespaces()
	assert.ElementsMatch(t, []string{"ns1", "ns2"}, namespaces)

	// Test without specific namespaces
	cfg = &config.ResourceConfig{
		Type: config.ResourceTypeConfigMap,
	}

	informer = NewResourceInformer(client, "", cfg, handler)
	namespaces = informer.getNamespaces()
	assert.Equal(t, []string{metav1.NamespaceAll}, namespaces)
}
