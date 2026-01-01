# Grafana Sidecar Example

This example demonstrates how to use `k8s-watcher` as a sidecar container to automatically sync Grafana dashboards from Kubernetes ConfigMaps.

## Overview

In this setup:
1. **Grafana** runs in a pod with a shared volume mounted at `/var/lib/grafana/dashboards`.
2. **k8s-watcher** runs as a sidecar in the same pod.
3. `k8s-watcher` watches for ConfigMaps with the label `dashboard=true`.
4. When a matching ConfigMap is found, `k8s-watcher` writes its data to the shared volume.
5. Grafana is configured to provision dashboards from that directory.

## Deployment

1. Apply the manifests:
   ```bash
   kubectl apply -f manifests.yaml
   ```

2. Port-forward to Grafana:
   ```bash
   kubectl port-forward svc/grafana 3000:3000
   ```

3. Open http://localhost:3000 (admin/admin). You should see the "Example Dashboard" in the Dashboards list.

## How it works

The `k8s-watcher` is configured via a ConfigMap to watch for resources:

```yaml
resources:
  type: configmap
  labels:
    - name: dashboard
      value: "true"
```

And output them to the shared folder:

```yaml
output:
  folder: /var/lib/grafana/dashboards
```
