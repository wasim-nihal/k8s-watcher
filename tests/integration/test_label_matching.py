"""Integration tests for label matching functionality."""

import time
import pytest
from kubernetes import client
from helpers import (
    create_configmap,
    create_secret,
    wait_for_file_in_pod,
    file_exists_in_pod
)


@pytest.mark.label
def test_exact_label_match(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test exact label matching (name + value)."""
    
    # Create ConfigMap with exact matching label
    cm = create_configmap(
        k8s_client,
        name="test-exact-match",
        namespace=test_namespace,
        labels={"app": "test"},  # Matches watcher config
        data={"file.txt": "matched"}
    )
    
    # File should be created
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-exact-match/file.txt"
    assert wait_for_file_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path,
        timeout=30
    )


@pytest.mark.label
def test_label_value_mismatch(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that resources with wrong label value are ignored."""
    
    # Create ConfigMap with wrong label value
    cm = create_configmap(
        k8s_client,
        name="test-wrong-value",
        namespace=test_namespace,
        labels={"app": "production"},  # "app" key exists but value doesn't match
        data={"file.txt": "should not sync"}
    )
    
    time.sleep(5)
    
    # File should NOT be created
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-wrong-value/file.txt"
    exists = file_exists_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path
    )
    assert not exists, "File should not have been created for mismatched label value"


@pytest.mark.label
def test_label_key_missing(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that resources without the required label key are ignored."""
    
    # Create ConfigMap without the required label key
    cm = create_configmap(
        k8s_client,
        name="test-missing-key",
        namespace=test_namespace,
        labels={"environment": "dev"},  # Different label key
        data={"file.txt": "should not sync"}
    )
    
    time.sleep(5)
    
    # File should NOT be created
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-missing-key/file.txt"
    exists = file_exists_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path
    )
    assert not exists, "File should not have been created for missing label key"


@pytest.mark.label
def test_multiple_labels_on_resource(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test resource with multiple labels where one matches."""
    
    # Create ConfigMap with multiple labels
    cm = create_configmap(
        k8s_client,
        name="test-multi-labels",
        namespace=test_namespace,
        labels={
            "app": "test",  # This matches
            "environment": "staging",
            "team": "platform"
        },
        data={"file.txt": "matched with multiple labels"}
    )
    
    # File should be created because "app: test" matches
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-multi-labels/file.txt"
    assert wait_for_file_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path,
        timeout=30
    )


@pytest.mark.label
def test_both_configmap_and_secret(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that both ConfigMaps and Secrets are watched when type is 'both'."""
    
    # Create ConfigMap
    cm = create_configmap(
        k8s_client,
        name="test-both-cm",
        namespace=test_namespace,
        labels={"app": "test"},
        data={"config.txt": "from configmap"}
    )
    
    # Create Secret
    secret = create_secret(
        k8s_client,
        name="test-both-secret",
        namespace=test_namespace,
        labels={"app": "test"},
        data={"secret.txt": b"from secret"}
    )
    
    # Both files should be created
    cm_file = f"/tmp/k8s-watcher-data/{test_namespace}/test-both-cm/config.txt"
    secret_file = f"/tmp/k8s-watcher-data/{test_namespace}/test-both-secret/secret.txt"
    
    assert wait_for_file_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        cm_file,
        timeout=30
    ), "ConfigMap file was not created"
    
    assert wait_for_file_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        secret_file,
        timeout=30
    ), "Secret file was not created"


@pytest.mark.label
def test_no_labels_on_resource(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that resources with no labels are ignored."""
    
    # Create ConfigMap with no labels
    cm = create_configmap(
        k8s_client,
        name="test-no-labels",
        namespace=test_namespace,
        labels={},  # Empty labels
        data={"file.txt": "should not sync"}
    )
    
    time.sleep(5)
    
    # File should NOT be created
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-no-labels/file.txt"
    exists = file_exists_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path
    )
    assert not exists, "File should not have been created for resource with no labels"


@pytest.mark.label
def test_case_sensitive_labels(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that label matching is case-sensitive."""
    
    # Create ConfigMap with different case
    cm = create_configmap(
        k8s_client,
        name="test-case-sensitive",
        namespace=test_namespace,
        labels={"app": "Test"},  # Capital T
        data={"file.txt": "should not sync"}
    )
    
    time.sleep(5)
    
    # File should NOT be created (case mismatch)
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-case-sensitive/file.txt"
    exists = file_exists_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path
    )
    assert not exists, "Label matching should be case-sensitive"
