"""Integration tests for ConfigMap watching functionality."""

import time
import pytest
from kubernetes import client
from helpers import (
    create_configmap,
    wait_for_file_in_pod,
    read_file_from_pod,
    file_exists_in_pod
)


@pytest.mark.configmap
def test_configmap_basic_sync(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that ConfigMap data is synced to filesystem."""
    
    # Create ConfigMap with matching labels
    cm = create_configmap(
        k8s_client,
        name="test-cm",
        namespace=test_namespace,
        labels={"app": "test"},
        data={"config.txt": "test content"}
    )
    
    # Wait for file to appear in watcher pod
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-cm/config.txt"
    assert wait_for_file_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path,
        timeout=30
    ), f"File {file_path} was not created in time"
    
    # Verify file content
    content = read_file_from_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path
    )
    assert content == "test content", f"Expected 'test content', got '{content}'"


@pytest.mark.configmap
def test_configmap_update(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that ConfigMap updates are reflected in files."""
    
    # Create initial ConfigMap
    cm = create_configmap(
        k8s_client,
        name="test-cm-update",
        namespace=test_namespace,
        labels={"app": "test"},
        data={"data.txt": "initial content"}
    )
    
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-cm-update/data.txt"
    
    # Wait for initial file
    assert wait_for_file_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path,
        timeout=30
    )
    
    # Update ConfigMap
    cm.data["data.txt"] = "updated content"
    k8s_client.patch_namespaced_config_map(
        name="test-cm-update",
        namespace=test_namespace,
        body=cm
    )
    
    # Wait for update to propagate
    time.sleep(5)
    
    # Verify updated content
    content = read_file_from_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path
    )
    assert content == "updated content", f"Expected 'updated content', got '{content}'"


@pytest.mark.configmap
def test_configmap_multiple_keys(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that all keys in a ConfigMap are synced."""
    
    # Create ConfigMap with multiple keys
    cm = create_configmap(
        k8s_client,
        name="test-cm-multi",
        namespace=test_namespace,
        labels={"app": "test"},
        data={
            "file1.txt": "content1",
            "file2.txt": "content2",
            "file3.txt": "content3"
        }
    )
    
    # Verify all files are created
    base_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-cm-multi"
    
    for filename, expected_content in [
        ("file1.txt", "content1"),
        ("file2.txt", "content2"),
        ("file3.txt", "content3")
    ]:
        file_path = f"{base_path}/{filename}"
        
        assert wait_for_file_in_pod(
            k8s_client,
            watcher_deployment["pod_name"],
            test_namespace,
            file_path,
            timeout=30
        ), f"File {file_path} was not created"
        
        content = read_file_from_pod(
            k8s_client,
            watcher_deployment["pod_name"],
            test_namespace,
            file_path
        )
        assert content == expected_content


@pytest.mark.configmap
def test_configmap_annotation_override(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that folder annotation overrides default output path."""
    
    # Create ConfigMap with folder annotation
    cm = create_configmap(
        k8s_client,
        name="test-cm-annotation",
        namespace=test_namespace,
        labels={"app": "test"},
        annotations={"k8s-watcher-target-dir": "/tmp/k8s-watcher-data/custom-path"},
        data={"custom.txt": "custom content"}
    )
    
    # File should be at the custom path
    file_path = "/tmp/k8s-watcher-data/custom-path/custom.txt"
    
    assert wait_for_file_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path,
        timeout=30
    ), f"File {file_path} was not created at custom path"
    
    content = read_file_from_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path
    )
    assert content == "custom content"


@pytest.mark.configmap
def test_configmap_ignored_without_labels(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that ConfigMaps without matching labels are ignored."""
    
    # Create ConfigMap without matching labels
    cm = create_configmap(
        k8s_client,
        name="test-cm-ignored",
        namespace=test_namespace,
        labels={"app": "other"},  # Different label value
        data={"ignored.txt": "should not be synced"}
    )
    
    # Wait a bit to see if file appears (it shouldn't)
    time.sleep(5)
    
    # Verify file was NOT created
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-cm-ignored/ignored.txt"
    
    exists = file_exists_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path
    )
    
    assert not exists, f"File {file_path} should not have been created"


@pytest.mark.configmap
def test_configmap_deletion(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test behavior when ConfigMap is deleted."""
    
    # Create ConfigMap
    cm = create_configmap(
        k8s_client,
        name="test-cm-delete",
        namespace=test_namespace,
        labels={"app": "test"},
        data={"delete-test.txt": "test content"}
    )
    
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-cm-delete/delete-test.txt"
    
    # Wait for file to be created
    assert wait_for_file_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path,
        timeout=30
    )
    
    # Delete ConfigMap
    k8s_client.delete_namespaced_config_map(
        name="test-cm-delete",
        namespace=test_namespace
    )
    
    # Note: In the current implementation, files are not deleted when ConfigMaps are deleted
    # This test documents the current behavior - files persist after deletion
    time.sleep(5)
    
    # File should still exist (current behavior)
    exists = file_exists_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path
    )
    
    assert exists, "File persists after ConfigMap deletion (current behavior)"
