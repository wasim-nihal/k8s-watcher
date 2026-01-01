"""Integration test for Grafana sidecar example."""

import time
import pytest
import yaml
import os
from kubernetes import client
from helpers import wait_for_pod_ready, get_pod_logs, wait_for_file_in_pod

@pytest.mark.example
def test_grafana_sidecar_example(
    kind_cluster: str,
    docker_image: str,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """
    Test the Grafana sidecar example by deploying the manifests
    and verifying the dashboard file is synced.
    """
    # Read the example manifests
    manifest_path = os.path.join(
        os.path.dirname(__file__), 
        "../../examples/grafana-sidecar/manifests.yaml"
    )
    
    with open(manifest_path, 'r') as f:
        manifests = list(yaml.safe_load_all(f))
    
    # Apply manifests to the test namespace
    # We need to patch the image to use the locally built one
    # and update the namespace
    
    print(f"\nDeploying Grafana example to namespace: {test_namespace}")
    
    apps_v1 = client.AppsV1Api()
    rbac_v1 = client.RbacAuthorizationV1Api()
    
    for manifest in manifests:
        if manifest is None:
            continue
            
        # Update namespace
        if "metadata" in manifest:
            manifest["metadata"]["namespace"] = test_namespace
            
        # Handle different kinds
        kind = manifest["kind"]
        
        if kind == "ConfigMap":
            # Patch watcher-config to use the correct namespace in config.yaml
            if manifest["metadata"]["name"] == "watcher-config":
                config_content = manifest["data"]["config.yaml"]
                manifest["data"]["config.yaml"] = config_content.replace(
                    "namespace: default",
                    f"namespace: {test_namespace}"
                )
            
            k8s_client.create_namespaced_config_map(
                namespace=test_namespace,
                body=manifest
            )
        elif kind == "Service":
            k8s_client.create_namespaced_service(
                namespace=test_namespace,
                body=manifest
            )
        elif kind == "ServiceAccount":
            k8s_client.create_namespaced_service_account(
                namespace=test_namespace,
                body=manifest
            )
        elif kind == "Role":
            rbac_v1.create_namespaced_role(
                namespace=test_namespace,
                body=manifest
            )
        elif kind == "RoleBinding":
            # Update subject namespace
            for subject in manifest.get("subjects", []):
                subject["namespace"] = test_namespace
            rbac_v1.create_namespaced_role_binding(
                namespace=test_namespace,
                body=manifest
            )
        elif kind == "Deployment":
            # Patch image
            for container in manifest["spec"]["template"]["spec"]["containers"]:
                if container["name"] == "k8s-watcher":
                    container["image"] = docker_image
                    container["imagePullPolicy"] = "Never"
            
            apps_v1.create_namespaced_deployment(
                namespace=test_namespace,
                body=manifest
            )

    # Wait for Grafana pod
    print("Waiting for Grafana pod to be ready...")
    
    # Give it a moment to be created
    time.sleep(2)
    
    pods = k8s_client.list_namespaced_pod(
        namespace=test_namespace,
        label_selector="app=grafana"
    )
    
    if not pods.items:
        raise RuntimeError("Grafana pod not found")
    
    pod_name = pods.items[0].metadata.name
    
    if not wait_for_pod_ready(k8s_client, pod_name, test_namespace, timeout=120):
        logs = get_pod_logs(k8s_client, pod_name, test_namespace, container="k8s-watcher")
        print(f"Watcher logs:\n{logs}")
        raise RuntimeError(f"Pod {pod_name} did not become ready")
        
    print(f"Pod {pod_name} is ready")
    
    # Verify dashboard file exists in shared volume
    # The example config writes to /var/lib/grafana/dashboards
    # and the dashboard ConfigMap has key "simple-dashboard.json"
    # With uniqueFilenames: true, it might be in a subdirectory or just the file
    # The example config has:
    # output:
    #   folder: /var/lib/grafana/dashboards
    #   uniqueFilenames: true
    #
    # If uniqueFilenames is true, it usually creates <namespace>/<name>/<key>
    # Let's check the watcher logs to see where it wrote
    
    time.sleep(5) # Wait for sync
    
    # Check if file exists. 
    # Based on logs, with uniqueFilenames: true, it creates:
    # /var/lib/grafana/dashboards/<namespace>/<configmap-name>-<key>
    expected_path = f"/var/lib/grafana/dashboards/{test_namespace}/example-dashboard-simple-dashboard.json"
    
    print(f"Verifying file exists at: {expected_path}")
    if wait_for_file_in_pod(k8s_client, pod_name, test_namespace, expected_path, container="grafana"):
        print("Dashboard file found!")
    else:
        # Debugging
        print("Dashboard file not found. Checking logs...")
        logs = get_pod_logs(k8s_client, pod_name, test_namespace, container="k8s-watcher")
        print(f"Watcher logs:\n{logs}")
        
        # List files in directory
        from kubernetes.stream import stream
        exec_command = ["ls", "-R", "/var/lib/grafana/dashboards"]
        resp = stream(k8s_client.connect_get_namespaced_pod_exec,
                    pod_name,
                    test_namespace,
                    command=exec_command,
                    stderr=True, stdin=False,
                    stdout=True, tty=False,
                    container="grafana")
        print(f"Files in /var/lib/grafana/dashboards:\n{resp}")
        
        pytest.fail("Dashboard file not synced")
