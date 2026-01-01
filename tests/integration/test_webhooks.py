"""Integration tests for webhook notification functionality."""

import time
import pytest
from kubernetes import client
from helpers import (
    create_configmap,
    wait_for_file_in_pod,
    get_pod_logs
)


@pytest.mark.webhook
def test_webhook_called_on_configmap_create(
    webhook_watcher_deployment: dict,
    mock_webhook_server: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that webhook is called when ConfigMap is created."""
    
    # Create ConfigMap with matching labels
    configmap = client.V1ConfigMap(
        metadata=client.V1ObjectMeta(
            name="test-webhook-cm",
            labels={"app": "webhook-test"}
        ),
        data={"test.txt": "webhook test content"}
    )
    
    k8s_client.create_namespaced_config_map(
        namespace=test_namespace,
        body=configmap
    )
    
    # Wait for watcher to process
    time.sleep(5)
    
    # Verify file was created (confirms watcher processed the resource)
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-webhook-cm/test.txt"
    assert wait_for_file_in_pod(
        k8s_client,
        webhook_watcher_deployment["pod_name"],
        test_namespace,
        file_path,
        timeout=30
    ), f"File {file_path} was not created"
    
    # Check watcher logs for successful webhook call
    logs = get_pod_logs(
        k8s_client,
        webhook_watcher_deployment["pod_name"],
        test_namespace,
        tail_lines=50
    )
    
    # Verify webhook was called successfully
    assert "Request completed successfully" in logs, \
        f"Webhook call not found in logs. Logs:\n{logs}"


@pytest.mark.webhook
def test_webhook_payload_content(
    webhook_watcher_deployment: dict,
    mock_webhook_server: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that webhook receives correct payload with resource info."""
    
    resource_name = "test-webhook-payload"
    
    # Create ConfigMap
    configmap = client.V1ConfigMap(
        metadata=client.V1ObjectMeta(
            name=resource_name,
            labels={"app": "webhook-test"}
        ),
        data={"payload.txt": "payload test"}
    )
    
    k8s_client.create_namespaced_config_map(
        namespace=test_namespace,
        body=configmap
    )
    
    # Wait for processing
    time.sleep(5)
    
    # Check logs for the resource name and namespace in the request
    logs = get_pod_logs(
        k8s_client,
        webhook_watcher_deployment["pod_name"],
        test_namespace,
        tail_lines=50
    )
    
    # Verify the processing log shows the resource was handled
    assert "Processing resource" in logs, \
        f"Resource processing not found in logs. Logs:\n{logs}"
    assert resource_name in logs, \
        f"Resource name '{resource_name}' not found in logs. Logs:\n{logs}"


@pytest.mark.webhook
def test_webhook_multiple_resources(
    webhook_watcher_deployment: dict,
    mock_webhook_server: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that webhook is called for multiple resources."""
    
    # Create multiple ConfigMaps
    for i in range(3):
        configmap = client.V1ConfigMap(
            metadata=client.V1ObjectMeta(
                name=f"test-multi-webhook-{i}",
                labels={"app": "webhook-test"}
            ),
            data={f"file{i}.txt": f"content {i}"}
        )
        
        k8s_client.create_namespaced_config_map(
            namespace=test_namespace,
            body=configmap
        )
    
    # Wait for all to be processed
    time.sleep(10)
    
    # Check logs for all resources
    logs = get_pod_logs(
        k8s_client,
        webhook_watcher_deployment["pod_name"],
        test_namespace,
        tail_lines=100
    )
    
    # Verify all resources were processed
    for i in range(3):
        assert f"test-multi-webhook-{i}" in logs, \
            f"Resource test-multi-webhook-{i} not found in logs"
    
    # Count successful webhook calls
    webhook_success_count = logs.count("Request completed successfully")
    assert webhook_success_count >= 3, \
        f"Expected at least 3 webhook calls, got {webhook_success_count}"


@pytest.mark.webhook
def test_webhook_basic_auth(
    webhook_watcher_deployment_auth: dict,
    mock_webhook_server_auth: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that webhook calls with basic authentication succeed."""
    
    # Create ConfigMap with matching labels for auth test
    configmap = client.V1ConfigMap(
        metadata=client.V1ObjectMeta(
            name="test-webhook-auth",
            labels={"app": "webhook-auth-test"}
        ),
        data={"auth-test.txt": "authenticated content"}
    )
    
    k8s_client.create_namespaced_config_map(
        namespace=test_namespace,
        body=configmap
    )
    
    # Wait for processing
    time.sleep(5)
    
    # Check logs for successful authenticated webhook call
    logs = get_pod_logs(
        k8s_client,
        webhook_watcher_deployment_auth["pod_name"],
        test_namespace,
        tail_lines=50
    )
    
    # Verify webhook was called successfully with auth
    assert "Request completed successfully" in logs, \
        f"Authenticated webhook call not found in logs. Logs:\n{logs}"

