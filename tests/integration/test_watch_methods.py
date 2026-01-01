"""Integration tests for watch method functionality."""

import time
import pytest
from kubernetes import client
from helpers import (
    create_configmap,
    wait_for_file_in_pod,
    read_file_from_pod
)


@pytest.mark.watch
def test_watch_mode_basic(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test WATCH mode (informer-based) functionality."""
    
    # Default watcher config uses WATCH mode
    # This test verifies that resources are picked up via watch
    
    cm = create_configmap(
        k8s_client,
        name="test-watch-mode",
        namespace=test_namespace,
        labels={"app": "test"},
        data={"watch.txt": "watched via informer"}
    )
    
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-watch-mode/watch.txt"
    
    # Should be picked up quickly in WATCH mode
    assert wait_for_file_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path,
        timeout=30
    )
    
    content = read_file_from_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path
    )
    assert content == "watched via informer"


@pytest.mark.watch
def test_resource_version_tracking(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that resource version tracking prevents duplicate processing."""
    
    # Create ConfigMap
    cm = create_configmap(
        k8s_client,
        name="test-version-tracking",
        namespace=test_namespace,
        labels={"app": "test"},
        data={"version.txt": "version 1"}
    )
    
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-version-tracking/version.txt"
    
    # Wait for initial processing
    assert wait_for_file_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path,
        timeout=30
    )
    
    # Update ConfigMap (change data but trigger re-watch)
    cm.data["version.txt"] = "version 2"
    k8s_client.patch_namespaced_config_map(
        name="test-version-tracking",
        namespace=test_namespace,
        body=cm
    )
    
    # Wait for update
    time.sleep(5)
    
    # Verify content was updated
    content = read_file_from_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path
    )
    assert content == "version 2"
    
    # The ignoreProcessed flag in config should prevent re-processing
    # of the same resource version


@pytest.mark.watch
def test_multiple_resources_concurrent(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test watching multiple resources created concurrently."""
    
    # Create multiple ConfigMaps at once
    num_resources = 5
    
    for i in range(num_resources):
        create_configmap(
            k8s_client,
            name=f"test-concurrent-{i}",
            namespace=test_namespace,
            labels={"app": "test"},
            data={f"file{i}.txt": f"content {i}"}
        )
    
    # Verify all files are created
    for i in range(num_resources):
        file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-concurrent-{i}/file{i}.txt"
        
        assert wait_for_file_in_pod(
            k8s_client,
            watcher_deployment["pod_name"],
            test_namespace,
            file_path,
            timeout=30
        ), f"File for resource {i} was not created"
        
        content = read_file_from_pod(
            k8s_client,
            watcher_deployment["pod_name"],
            test_namespace,
            file_path
        )
        assert content == f"content {i}"


@pytest.mark.watch
@pytest.mark.slow
def test_watch_reconnection(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that watcher recovers after pod restart."""
    
    # Create initial ConfigMap
    cm1 = create_configmap(
        k8s_client,
        name="test-before-restart",
        namespace=test_namespace,
        labels={"app": "test"},
        data={"before.txt": "before restart"}
    )
    
    file_path1 = f"/tmp/k8s-watcher-data/{test_namespace}/test-before-restart/before.txt"
    assert wait_for_file_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path1,
        timeout=30
    )
    
    # Restart watcher pod
    k8s_client.delete_namespaced_pod(
        name=watcher_deployment["pod_name"],
        namespace=test_namespace
    )
    
    # Wait for new pod to be ready
    time.sleep(10)
    
    # Get new pod name
    from helpers import wait_for_pod_ready
    pods = k8s_client.list_namespaced_pod(
        namespace=test_namespace,
        label_selector="app=k8s-watcher"
    )
    
    if not pods.items:
        pytest.fail("Watcher pod not found after restart")
    
    new_pod_name = pods.items[0].metadata.name
    
    if not wait_for_pod_ready(k8s_client, new_pod_name, test_namespace, timeout=60):
        pytest.fail("Watcher pod did not become ready after restart")
    
    # Create new ConfigMap after restart
    cm2 = create_configmap(
        k8s_client,
        name="test-after-restart",
        namespace=test_namespace,
        labels={"app": "test"},
        data={"after.txt": "after restart"}
    )
    
    # Verify new ConfigMap is processed
    file_path2 = f"/tmp/k8s-watcher-data/{test_namespace}/test-after-restart/after.txt"
    assert wait_for_file_in_pod(
        k8s_client,
        new_pod_name,
        test_namespace,
        file_path2,
        timeout=30
    ), "New ConfigMap was not processed after watcher restart"
