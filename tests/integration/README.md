# Integration Tests for k8s-watcher

This directory contains integration tests for the k8s-watcher application using pytest and Kubernetes in Docker (KinD).

## Overview

The integration tests validate k8s-watcher's functionality in a real Kubernetes environment by:
- Creating a KinD cluster
- Building and deploying k8s-watcher
- Creating test ConfigMaps and Secrets
- Verifying file synchronization and actions

## Prerequisites

- **Docker**: For running KinD clusters
- **KinD**: Kubernetes in Docker (`go install sigs.k8s.io/kind@latest`)
- **kubectl**: Kubernetes CLI (`brew install kubectl` or download from kubernetes.io)
- **Python 3.11+**: For running pytest
- **pip**: Python package manager

## Installation

```bash
# Install Python dependencies
cd tests/integration
pip install -r requirements.txt
```

## Running Tests

### Run All Tests

```bash
cd tests/integration
pytest -v
```

### Run Specific Test Suite

```bash
# ConfigMap tests
pytest test_configmap_watch.py -v

# Secret tests
pytest test_secret_watch.py -v

# Label matching tests
pytest test_label_matching.py -v

# Watch method tests
pytest test_watch_methods.py -v
```

### Run Tests by Marker

```bash
# Run only ConfigMap tests
pytest -m configmap -v

# Run only Secret tests
pytest -m secret -v

# Skip slow tests
pytest -m "not slow" -v
```

### Run with Coverage

```bash
pytest --cov=. --cov-report=html
```

## Test Structure

```
tests/integration/
├── conftest.py              # Pytest fixtures (cluster, deployment, etc.)
├── helpers.py               # Helper utilities for tests
├── pytest.ini               # Pytest configuration
├── requirements.txt         # Python dependencies
├── kind-config.yaml         # KinD cluster configuration
│
├── test_configmap_watch.py  # ConfigMap watching tests
├── test_secret_watch.py     # Secret watching tests
├── test_label_matching.py   # Label selector tests
├── test_webhooks.py         # Webhook notification tests
├── test_scripts.py          # Script execution tests
└── test_watch_methods.py    # Watch mechanism tests
```

## Test Coverage

### ConfigMap Tests (7 tests)
- Basic sync
- Updates
- Multiple keys
- Annotation-based path override
- Label filtering
- Resource deletion behavior

### Secret Tests (7 tests)
- Basic sync
- Updates
- Binary data handling
- Multiple keys
- Label filtering
- TLS certificates

### Label Matching Tests (7 tests)
- Exact label matching
- Label value mismatches
- Missing label keys
- Multiple labels
- Type 'both' behavior
- Case sensitivity

### Watch Method Tests (5 tests)
- WATCH mode functionality
- Resource version tracking
- Concurrent updates
- Watch reconnection

### Webhook Tests (4 tests - placeholder)
- Webhook calls on changes
- Payload content
- Retry logic
- Authentication

### Script Tests (3 tests - placeholder)
- Script execution
- Timeout handling
- Failure handling

## Fixtures

### Session-Scoped Fixtures

- **`kind_cluster`**: Creates/manages KinD cluster for all tests
- **`docker_image`**: Builds and loads k8s-watcher Docker image
- **`k8s_client`**: Kubernetes API client

### Function-Scoped Fixtures

- **`test_namespace`**: Isolated namespace for each test
- **`watcher_deployment`**: Deploys k8s-watcher with RBAC
- **`watcher_config_basic`**: Basic watcher configuration
- **`webhook_server`**: Mock HTTP server for webhooks

## Debugging

### View Pod Logs

```bash
# Get test namespace
kubectl get namespaces | grep test-

# View watcher logs
kubectl logs -n <test-namespace> deployment/k8s-watcher
```

### Keep KinD Cluster After Tests

Comment out the cleanup code in `conftest.py`:

```python
# yield cluster_name
# Comment out these lines:
# print(f"\n🗑️  Deleting KinD cluster: {cluster_name}")
# subprocess.run(["kind", "delete", "cluster", "--name", cluster_name])
```

### Run Tests with Debug Logging

```bash
pytest -v --log-cli-level=DEBUG
```

### Access Test Cluster

```bash
# Set kubectl context
kubectl config use-context kind-k8s-watcher-test

# List all resources
kubectl get all --all-namespaces
```

## CI/CD

Integration tests run automatically in GitHub Actions on:
- Pull requests
- Pushes to main branch
- Manual workflow dispatch

See `.github/workflows/integration-tests.yml` for the full workflow.

## Known Limitations

1. **Webhook Tests**: Currently placeholders - require custom fixture to deploy webhook server in cluster
2. **Script Tests**: Currently placeholders - require mounting test scripts into watcher pod
3. **LIST Mode**: Not currently tested (only WATCH mode)
4. **File Deletion**: Watcher doesn't delete files when resources are removed

## Contributing

When adding new tests:

1. Use appropriate pytest markers (`@pytest.mark.configmap`, etc.)
2. Use descriptive test names (`test_<what>_<condition>`)
3. Clean up resources in test or rely on namespace deletion
4. Add docstrings explaining what the test validates
5. Keep tests independent and order-agnostic

## Troubleshooting

### KinD Cluster Creation Fails

```bash
# Delete existing cluster
kind delete cluster --name k8s-watcher-test

# Try creating again
kind create cluster --config kind-config.yaml --name k8s-watcher-test
```

### Docker Image Not Found

```bash
# Rebuild image
docker build -t k8s-watcher:test ../..

# Load into KinD
kind load docker-image k8s-watcher:test --name k8s-watcher-test
```

### Watcher Pod Not Starting

```bash
# Check pod status
kubectl get pods -n <test-namespace>

# View pod events
kubectl describe pod -n <test-namespace> <pod-name>

# Check logs
kubectl logs -n <test-namespace> <pod-name>
```

### Tests Timing Out

- Increase timeout values in test code
- Check if KinD cluster has sufficient resources
- Verify network connectivity

## Performance

Typical test execution times:
- Cluster creation: ~30 seconds (first time)
- Docker image build/load: ~30 seconds
- Per-test execution: ~5-10 seconds
- Total for all tests: ~5-10 minutes

## License

Same as main project - Apache License 2.0
