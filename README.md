# k8s-watcher

[![Go Report Card](https://goreportcard.com/badge/github.com/wasim-nihal/k8s-watcher)](https://goreportcard.com/report/github.com/wasim-nihal/k8s-watcher)
[![GoDoc](https://godoc.org/github.com/wasim-nihal/k8s-watcher?status.svg)](https://godoc.org/github.com/wasim-nihal/k8s-watcher)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

![Banner](img/K8s-Watcher.png)


k8s-watcher is a powerful and flexible Kubernetes sidecar application that monitors ConfigMaps and Secrets, automatically syncing their contents to the filesystem and triggering custom actions on changes.

## Features

- Watch multiple ConfigMaps and Secrets simultaneously
- Filter resources using label selectors
- Flexible file output management
- Support for custom actions on changes (webhooks, scripts)
- Secure handling of sensitive data
- Efficient resource processing with version tracking
- Comprehensive logging options

## Installation

```bash
go install github.com/wasim-nihal/k8s-watcher@latest
```

Or build from source:

```bash
git clone https://github.com/wasim-nihal/k8s-watcher.git
cd k8s-watcher
go build ./cmd/k8s-watcher
```

## Quick Start

1. Create a configuration file:

```yaml
output:
  folder: /data/configs
  uniqueFilenames: true

kubernetes:
  namespace: default

resources:
  type: configmap
  method: WATCH
  labels:
    - name: app
      value: myapp
```

2. Run k8s-watcher:

```bash
k8s-watcher -config config.yaml
```

## Configuration

### Output Settings

```yaml
output:
  folder: string              # Base output directory (required)
  folderAnnotation: string    # Annotation key to override output location
  uniqueFilenames: boolean    # Generate unique filenames for duplicate keys
  defaultFileMode: string     # Default file permissions (e.g., "440")
```

### Kubernetes Settings

```yaml
kubernetes:
  kubeconfig: string         # Path to kubeconfig file
  namespace: string          # Namespace(s) to watch
  skipTLSVerify: boolean    # Skip TLS verification
```

### Resource Settings

```yaml
resources:
  type: string              # configmap/secret/both
  method: string           # WATCH/LIST/SLEEP
  watchConfig:
    serverTimeout: int     # Server-side timeout
    clientTimeout: int     # Client-side timeout
    errorThrottleTime: int # Error throttle time
    ignoreProcessed: bool  # Skip already processed versions
  labels:
    - name: string         # Label name
      value: string        # Label value
      script:
        path: string       # Script to execute
        timeout: int       # Script timeout
      request:
        url: string        # Webhook URL
        method: string     # HTTP method
        retry:
          total: int       # Total retries
          backoffFactor: float # Retry backoff
```

## Use Cases

### 1. Configuration Sync

Sync application configurations from ConfigMaps:

```yaml
resources:
  type: configmap
  labels:
    - name: app
      value: myapp
```

### 2. Secret Management

Monitor and sync secrets with secure permissions:

```yaml
output:
  defaultFileMode: "400"
resources:
  type: secret
  labels:
    - name: type
      value: credentials
```

### 3. Multi-Resource Watching

Watch both ConfigMaps and Secrets:

```yaml
resources:
  type: both
  labels:
    - name: environment
      value: production
```

## Kubernetes Integration

Example deployment as a sidecar:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  template:
    spec:
      containers:
        - name: main-app
          image: myapp:latest
          volumeMounts:
            - name: config-volume
              mountPath: /config
        
        - name: k8s-watcher
          image: k8s-watcher:latest
          volumeMounts:
            - name: config-volume
              mountPath: /config
            - name: watcher-config
              mountPath: /etc/k8s-watcher
      
      volumes:
        - name: config-volume
          emptyDir: {}
        - name: watcher-config
          configMap:
            name: watcher-config
```

## Development

### Prerequisites

- Go 1.22 or later
- Access to a Kubernetes cluster
- Make (optional, for using Makefile)

### Building

```bash
make build
```

### Testing

```bash
make test
```

### Running Tests with Coverage

```bash
make coverage
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [client-go](https://github.com/kubernetes/client-go)
