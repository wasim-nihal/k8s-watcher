# k8s-watcher Implementation Guide

This document provides a detailed guide for implementing the k8s-watcher application in Go, following the design specified in `design.md`.

## Table of Contents
1. [Project Structure](#project-structure)
2. [Core Components Implementation](#core-components-implementation)
3. [Configuration Management](#configuration-management)
4. [Kubernetes Integration](#kubernetes-integration)
5. [Resource Watching](#resource-watching)
6. [File Operations](#file-operations)
7. [HTTP Client Implementation](#http-client-implementation)
8. [Testing Strategy](#testing-strategy)
9. [Building and Deployment](#building-and-deployment)

## Project Structure

```plaintext
.
├── cmd/
│   └── k8s-watcher/
│       └── main.go
├── pkg/
│   ├── config/
│   │   ├── types.go
│   │   └── loader.go
│   ├── watcher/
│   │   ├── watcher.go
│   │   ├── informer.go
│   │   └── handler.go
│   ├── label/
│   │   └── manager.go
│   ├── file/
│   │   └── handler.go
│   ├── http/
│   │   └── client.go
│   └── logger/
│       └── logger.go
├── internal/
│   └── utils/
│       ├── kubernetes.go
│       └── retry.go
├── test/
│   └── integration/
├── Dockerfile
├── go.mod
└── go.sum
```

## Core Components Implementation

### 1. Configuration Types (pkg/config/types.go)

```go
package config

type Config struct {
    Output     OutputConfig     `yaml:"output"`
    Kubernetes KubernetesConfig `yaml:"kubernetes"`
    Resources  ResourceConfig   `yaml:"resources"`
    Logging    LoggingConfig   `yaml:"logging"`
}

type OutputConfig struct {
    Folder           string `yaml:"folder"`
    FolderAnnotation string `yaml:"folderAnnotation"`
    UniqueFilenames  bool   `yaml:"uniqueFilenames"`
    DefaultFileMode  string `yaml:"defaultFileMode"`
}

type ResourceConfig struct {
    Type         string         `yaml:"type"`
    Method       string         `yaml:"method"`
    ResourceNames []string      `yaml:"resourceNames"`
    WatchConfig   WatchConfig   `yaml:"watchConfig"`
    Labels        []LabelConfig `yaml:"labels"`
}

type LabelConfig struct {
    Name    string         `yaml:"name"`
    Value   string         `yaml:"value"`
    Script  ScriptConfig   `yaml:"script"`
    Request RequestConfig  `yaml:"request"`
}

// Add other config structs...
```

### 2. Configuration Loader (pkg/config/loader.go)

```go
package config

import (
    "os"
    "path/filepath"
    "gopkg.in/yaml.v3"
)

type Loader struct {
    path string
}

func NewLoader(path string) *Loader {
    return &Loader{path: path}
}

func (l *Loader) Load() (*Config, error) {
    data, err := os.ReadFile(l.path)
    if err != nil {
        return nil, fmt.Errorf("reading config file: %w", err)
    }

    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("parsing config: %w", err)
    }

    if err := config.Validate(); err != nil {
        return nil, fmt.Errorf("validating config: %w", err)
    }

    return &config, nil
}
```

### 3. Resource Watcher (pkg/watcher/watcher.go)

```go
package watcher

import (
    "context"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/cache"
)

type ResourceWatcher struct {
    client        kubernetes.Interface
    labelManager  *label.Manager
    fileHandler   *file.Handler
    httpClient    *http.Client
    watchConfig   config.WatchConfig
}

func NewResourceWatcher(config *config.Config, client kubernetes.Interface) *ResourceWatcher {
    return &ResourceWatcher{
        client:       client,
        labelManager: label.NewManager(config.Resources.Labels),
        fileHandler:  file.NewHandler(config.Output),
        httpClient:   http.NewClient(config.Resources),
        watchConfig:  config.Resources.WatchConfig,
    }
}

func (w *ResourceWatcher) Start(ctx context.Context) error {
    // Initialize informers for ConfigMaps and/or Secrets
    // Set up event handlers
    // Start watching
    return nil
}
```

### 4. Label Manager (pkg/label/manager.go)

```go
package label

import (
    "k8s.io/apimachinery/pkg/labels"
    "k8s.io/apimachinery/pkg/selection"
)

type Manager struct {
    configs []config.LabelConfig
}

func NewManager(configs []config.LabelConfig) *Manager {
    return &Manager{configs: configs}
}

func (m *Manager) MatchLabels(resourceLabels map[string]string) []config.LabelConfig {
    var matches []config.LabelConfig
    
    for _, cfg := range m.configs {
        if m.matchLabel(cfg, resourceLabels) {
            matches = append(matches, cfg)
        }
    }
    
    return matches
}
```

### 5. File Handler (pkg/file/handler.go)

```go
package file

import (
    "os"
    "path/filepath"
    "strconv"
)

type Handler struct {
    config config.OutputConfig
}

func NewHandler(config config.OutputConfig) *Handler {
    return &Handler{config: config}
}

func (h *Handler) WriteFile(path string, data []byte) error {
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("creating directory: %w", err)
    }

    mode, err := h.getFileMode()
    if err != nil {
        return err
    }

    return os.WriteFile(path, data, mode)
}

func (h *Handler) getFileMode() (os.FileMode, error) {
    if h.config.DefaultFileMode == "" {
        return 0644, nil
    }
    
    mode, err := strconv.ParseUint(h.config.DefaultFileMode, 8, 32)
    if err != nil {
        return 0, fmt.Errorf("parsing file mode: %w", err)
    }
    
    return os.FileMode(mode), nil
}
```

### 6. HTTP Client (pkg/http/client.go)

```go
package http

import (
    "net/http"
    "time"
)

type Client struct {
    client  *http.Client
    config  config.ResourceConfig
}

func NewClient(config config.ResourceConfig) *Client {
    return &Client{
        client: &http.Client{
            Timeout: time.Second * 10,
            Transport: &http.Transport{
                TLSClientConfig: &tls.Config{
                    InsecureSkipVerify: config.Request.SkipTLSVerify,
                },
            },
        },
        config: config,
    }
}

func (c *Client) SendNotification(url string, method string, payload interface{}) error {
    // Implement notification logic with retry mechanism
    return nil
}
```

## Main Application (cmd/k8s-watcher/main.go)

```go
package main

import (
    "context"
    "flag"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/your-org/k8s-watcher/pkg/config"
    "github.com/your-org/k8s-watcher/pkg/watcher"
)

func main() {
    configPath := flag.String("config", "config.yaml", "Path to configuration file")
    flag.Parse()

    // Load configuration
    cfg, err := config.NewLoader(*configPath).Load()
    if err != nil {
        log.Fatalf("Loading config: %v", err)
    }

    // Initialize Kubernetes client
    client, err := kubernetes.NewForConfig(getKubeConfig(cfg))
    if err != nil {
        log.Fatalf("Creating Kubernetes client: %v", err)
    }

    // Create and start watcher
    w := watcher.NewResourceWatcher(cfg, client)
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle shutdown gracefully
    signalChan := make(chan os.Signal, 1)
    signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-signalChan
        cancel()
    }()

    if err := w.Start(ctx); err != nil {
        log.Fatalf("Starting watcher: %v", err)
    }
}
```

## Testing Strategy

### 1. Unit Tests

Create comprehensive unit tests for each package:

```go
// pkg/watcher/watcher_test.go
func TestResourceWatcher_HandleConfigMapUpdate(t *testing.T) {
    // Test cases
}

// pkg/label/manager_test.go
func TestManager_MatchLabels(t *testing.T) {
    // Test cases
}
```

### 2. Integration Tests

```go
// test/integration/watcher_test.go
func TestWatcherWithKubernetes(t *testing.T) {
    // Setup test cluster
    // Create test resources
    // Verify watcher behavior
}
```

## Building and Deployment

### Dockerfile

```dockerfile
FROM golang:1.22 AS builder

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o k8s-watcher ./cmd/k8s-watcher

FROM alpine:3.19
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/k8s-watcher /usr/local/bin/

ENTRYPOINT ["k8s-watcher"]
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-with-watcher
spec:
  template:
    spec:
      containers:
      - name: main-app
        image: your-app:latest
        volumeMounts:
        - name: shared-config
          mountPath: /config
      - name: k8s-watcher
        image: k8s-watcher:latest
        volumeMounts:
        - name: shared-config
          mountPath: /config
        - name: watcher-config
          mountPath: /etc/k8s-watcher
      volumes:
      - name: shared-config
        emptyDir: {}
      - name: watcher-config
        configMap:
          name: k8s-watcher-config
```

## Implementation Steps

1. Set up the project structure
2. Implement configuration management
3. Create the Kubernetes client wrapper
4. Implement the label manager
5. Create the file handler
6. Implement the HTTP client
7. Build the resource watcher
8. Write unit tests
9. Create integration tests
10. Set up CI/CD pipeline
11. Create Docker image
12. Write deployment manifests

## Development Workflow

1. Start with implementing the configuration package
2. Build and test each component independently
3. Integrate components gradually
4. Use interface-based design for testability
5. Add metrics and logging throughout
6. Implement graceful shutdown
7. Add health checks
8. Create documentation
9. Setup monitoring

## Best Practices

1. Use context for cancellation and timeouts
2. Implement proper error handling and logging
3. Add metrics for monitoring
4. Use interfaces for better testing
5. Follow Go best practices and idioms
6. Keep functions small and focused
7. Use meaningful variable and function names
8. Add comments for exported functions and types
9. Implement proper validation
10. Handle edge cases and errors gracefully
