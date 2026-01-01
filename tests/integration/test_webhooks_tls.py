"""Integration tests for webhook TLS functionality using cert-manager."""

import time
import pytest
from kubernetes import client
from helpers import (
    create_configmap,
    wait_for_file_in_pod,
    get_pod_logs
)


@pytest.mark.webhook
def test_webhook_tls_success(
    webhook_watcher_deployment_tls: dict,
    mock_webhook_server_tls: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that HTTPS webhook calls succeed with TLS."""
    
    # Create ConfigMap with matching labels for TLS test
    configmap = client.V1ConfigMap(
        metadata=client.V1ObjectMeta(
            name="test-webhook-tls",
            labels={"app": "webhook-tls-test"}
        ),
        data={"tls-test.txt": "tls encrypted content"}
    )
    
    k8s_client.create_namespaced_config_map(
        namespace=test_namespace,
        body=configmap
    )
    
    # Wait for processing
    time.sleep(5)
    
    # Verify file was created
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-webhook-tls/tls-test.txt"
    assert wait_for_file_in_pod(
        k8s_client,
        webhook_watcher_deployment_tls["pod_name"],
        test_namespace,
        file_path,
        timeout=30
    ), f"File {file_path} was not created"
    
    # Check logs for successful TLS webhook call
    logs = get_pod_logs(
        k8s_client,
        webhook_watcher_deployment_tls["pod_name"],
        test_namespace,
        tail_lines=50
    )
    
    # Verify HTTPS webhook was called successfully
    assert "Request completed successfully" in logs, \
        f"TLS webhook call not found in logs. Logs:\n{logs}"


@pytest.mark.webhook
def test_webhook_tls_multiple_resources(
    webhook_watcher_deployment_tls: dict,
    mock_webhook_server_tls: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that multiple HTTPS webhook calls work correctly."""
    
    # Create multiple ConfigMaps
    for i in range(2):
        configmap = client.V1ConfigMap(
            metadata=client.V1ObjectMeta(
                name=f"test-tls-multi-{i}",
                labels={"app": "webhook-tls-test"}
            ),
            data={f"tls-file{i}.txt": f"tls content {i}"}
        )
        
        k8s_client.create_namespaced_config_map(
            namespace=test_namespace,
            body=configmap
        )
    
    # Wait for processing
    time.sleep(10)
    
    # Check logs
    logs = get_pod_logs(
        k8s_client,
        webhook_watcher_deployment_tls["pod_name"],
        test_namespace,
        tail_lines=100
    )
    
    # Count successful TLS webhook calls
    webhook_success_count = logs.count("Request completed successfully")
    assert webhook_success_count >= 2, \
        f"Expected at least 2 TLS webhook calls, got {webhook_success_count}"
