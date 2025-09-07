# k8s-watcher Design Document

## Overview

k8s-watcher is a Kubernetes sidecar application written in Go that extends the functionality of the kiwigrid k8s-sidecar by allowing monitoring of multiple ConfigMaps and Secrets simultaneously. The application watches for changes in Kubernetes resources and can trigger various actions based on those changes.

## Key Features

1. Multi-Resource Monitoring
   - Watch multiple ConfigMaps and Secrets simultaneously
   - Filter resources based on multiple label selectors
   - Support for both single and multi-namespace monitoring

2. Flexible Output Management
   - Write resource contents to specified filesystem locations
   - Support for custom folder paths via annotations
   - Maintain directory structure based on configuration

3. Event Handling
   - HTTP webhook notifications for resource changes
   - Support for custom scripts execution
   - Basic authentication for HTTP requests
   - Configurable retry mechanisms

4. Resource Types Support
   - ConfigMaps
   - Secrets
   - Support for both data and binaryData fields
   - URL-based content fetching (for .url suffixed keys)

## Configuration Structure

The application uses YAML configuration with the following structure:

```yaml
output:
  folder: string              # Base output directory for all watched resources (required)
  folderAnnotation: string    # Annotation key to override output location (default: k8s-sidecar-target-directory)
  uniqueFilenames: boolean    # Generate unique filenames for duplicate keys (default: false)
  defaultFileMode: string     # Default file permissions (e.g., "440")

kubernetes:
  kubeconfig: string         # Path to kubeconfig file (optional, uses in-cluster config if not set)
  namespace: string          # Namespace(s) to watch (comma-separated, "ALL" for all namespaces)
  skipTLSVerify: boolean    # Skip TLS verification for K8s API calls

resources:
  type: string              # Resource type (configmap/secret/both)
  method: string           # Watch method (WATCH/LIST/SLEEP)
  resourceNames: []string  # Optional list of specific resources to watch
  watchConfig:
    serverTimeout: int     # Server-side watch timeout in seconds (default: 60)
    clientTimeout: int     # Client-side watch timeout in seconds (default: 66)
    errorThrottleTime: int # Time to wait after errors (default: 5)
    ignoreProcessed: bool  # Ignore already processed versions (default: false)
  labels:                  # Array of label configurations
    - name: string         # Label name to watch
      value: string        # Optional label value to filter
      script:             # Optional script configuration
        path: string      # Path to script to execute
        timeout: int      # Script execution timeout
      request:            # Optional HTTP request configuration
        url: string       # Webhook URL
        method: string    # HTTP method (GET/POST)
        payload: object   # Optional request payload
        timeout: float    # Request timeout in seconds (default: 10)
        retry:
          total: int      # Total retry attempts (default: 5)
          connect: int    # Connect retry attempts (default: 10)
          read: int       # Read retry attempts (default: 5)
          backoffFactor: float # Retry backoff factor (default: 1.1)
        auth:
          basic:
            username: string
            password: string
            encoding: string  # Auth encoding (default: latin1)
          usernameFile: string # Path to username file
          passwordFile: string # Path to password file
        skipTLSVerify: boolean # Skip TLS verification for requests

logging:
  level: string           # Log level (DEBUG/INFO/WARN/ERROR/CRITICAL)
  format: string         # Log format (JSON/LOGFMT)
  timezone: string       # Log timezone (LOCAL/UTC)
  configPath: string     # Custom logging config file path
```

## Technical Design

### 1. Core Components

#### Resource Watcher
- Implements Kubernetes informer pattern for efficient resource watching
- Maintains separate watchers for ConfigMaps and Secrets
- Handles resource events (Add/Update/Delete)

#### Label Manager
- Manages multiple label selector configurations
- Implements filtering logic for resources based on labels
- Supports dynamic label value matching

#### File Handler
- Manages file system operations
- Handles directory creation and file writing
- Supports atomic file operations for consistency

#### HTTP Client
- Manages webhook notifications
- Implements retry logic with exponential backoff
- Handles basic authentication

### 2. Event Flow

1. Application Startup
   ```
   Load Configuration → Validate Settings → Initialize Watchers
   ```

2. Resource Event Processing
   ```
   Resource Change → Label Filter → Process Content → Execute Actions
   ```

3. Action Execution
   ```
   Write Files → Execute Scripts → Send HTTP Notifications
   ```

## Implementation Details

### 1. Resource Watching

```go
type Watcher struct {
    clientset       kubernetes.Interface
    labelSelectors []LabelSelector
    eventHandler   EventHandler
}

type LabelSelector struct {
    Name   string
    Value  string
    Config ActionConfig
}

type ActionConfig struct {
    Script  string
    Request WebhookConfig
}
```

### 2. File Management

```go
type FileHandler struct {
    baseDir     string
    annotation  string
    permissions os.FileMode
}
```

### 3. HTTP Client

```go
type WebhookClient struct {
    client    *http.Client
    retryConf RetryConfig
}

type RetryConfig struct {
    MaxRetries      int
    BackoffFactor   float64
    MaxWaitSeconds  int
}
```

## Deployment

### Kubernetes Deployment Example

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
        # ... main application config ...
      - name: k8s-watcher
        image: k8s-watcher:latest
        volumeMounts:
        - name: shared-data
          mountPath: /data
        - name: config
          mountPath: /etc/k8s-watcher
      volumes:
      - name: shared-data
        emptyDir: {}
      - name: config
        configMap:
          name: k8s-watcher-config
```

## Configuration Examples

### Basic ConfigMap Watching
```yaml
output:
  folder: /data/configs
  defaultFileMode: "440"

kubernetes:
  namespace: default

resources:
  type: configmap
  method: WATCH
  labels:
    - name: app
      value: myapp
```

### Multi-Resource Watching with Advanced Features
```yaml
output:
  folder: /data/configs
  folderAnnotation: custom-target-dir
  uniqueFilenames: true
  defaultFileMode: "440"

kubernetes:
  namespace: "app-ns1,app-ns2"
  skipTLSVerify: false

resources:
  type: both
  method: WATCH
  watchConfig:
    serverTimeout: 60
    clientTimeout: 66
    errorThrottleTime: 5
  labels:
    - name: config-type
      value: database
      request:
        url: http://localhost:8080/reload
        method: POST
        retry:
          total: 5
          backoffFactor: 1.1
        auth:
          basic:
            username: "admin"
            password: "secret"
    - name: config-type
      value: cache
      script:
        path: /scripts/refresh-cache.sh
        timeout: 30
```

### URL Content Fetching Example
```yaml
output:
  folder: /data/binaries
  uniqueFilenames: true

kubernetes:
  namespace: default

resources:
  type: configmap
  method: WATCH
  labels:
    - name: binary-configs
      request:
        skipTLSVerify: true
        timeout: 30
        retry:
          total: 3
          connect: 5
```

## Future Enhancements

1. Support for CRD monitoring
2. Dynamic configuration reloading
3. Metrics exposition for Prometheus
4. Advanced filtering capabilities
5. Support for templating in output paths
6. Integration with external secret management systems

## Security Considerations

1. RBAC Configuration
   - Minimal required permissions
   - Namespace-scoped roles when possible

2. Secret Handling
   - Secure storage of sensitive data
   - Support for external secret providers

3. Authentication
   - Support for various authentication methods
   - Secure credential management
