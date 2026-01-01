"""Integration tests for Secret watching functionality."""

import time
import pytest
from kubernetes import client
from helpers import (
    create_secret,
    wait_for_file_in_pod,
    read_file_from_pod,
    file_exists_in_pod
)


@pytest.mark.secret
def test_secret_basic_sync(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that Secret data is synced to filesystem."""
    
    # Create Secret with matching labels
    secret = create_secret(
        k8s_client,
        name="test-secret",
        namespace=test_namespace,
        labels={"app": "test"},
        data={"credentials.txt": b"super secret data"}
    )
    
    # Wait for file to appear in watcher pod
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-secret/credentials.txt"
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
    assert content == "super secret data", f"Expected 'super secret data', got '{content}'"


@pytest.mark.secret
def test_secret_update(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that Secret updates are reflected in files."""
    
    # Create initial Secret
    secret = create_secret(
        k8s_client,
        name="test-secret-update",
        namespace=test_namespace,
        labels={"app": "test"},
        data={"password.txt": b"initial-password"}
    )
    
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-secret-update/password.txt"
    
    # Wait for initial file
    assert wait_for_file_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path,
        timeout=30
    )
    
    # Update Secret
    import base64
    secret.data["password.txt"] = base64.b64encode(b"updated-password").decode('utf-8')
    k8s_client.patch_namespaced_secret(
        name="test-secret-update",
        namespace=test_namespace,
        body=secret
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
    assert content == "updated-password", f"Expected 'updated-password', got '{content}'"


@pytest.mark.secret
def test_secret_binary_data(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that binary data in Secrets is handled correctly."""
    
    # Create Secret with binary data
    binary_data = bytes(range(256))  # All byte values 0-255
    
    secret = create_secret(
        k8s_client,
        name="test-secret-binary",
        namespace=test_namespace,
        labels={"app": "test"},
        data={"binary.dat": binary_data}
    )
    
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-secret-binary/binary.dat"
    
    assert wait_for_file_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path,
        timeout=30
    )
    
    # Read the file as binary and verify it matches
    # Note: The kubernetes stream API returns text, so we use base64 for binary comparison
    from kubernetes.stream import stream
    import base64
    exec_command = ['/bin/sh', '-c', f'base64 {file_path}']
    resp = stream(
        k8s_client.connect_get_namespaced_pod_exec,
        watcher_deployment["pod_name"],
        test_namespace,
        command=exec_command,
        stderr=False,
        stdin=False,
        stdout=True,
        tty=False,
    )
    
    # Decode base64 to get original binary content
    file_content = base64.b64decode(resp.strip())
    
    assert file_content == binary_data, "Binary data mismatch"


@pytest.mark.secret
def test_secret_multiple_keys(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that all keys in a Secret are synced."""
    
    # Create Secret with multiple keys
    secret = create_secret(
        k8s_client,
        name="test-secret-multi",
        namespace=test_namespace,
        labels={"app": "test"},
        data={
            "username": b"admin",
            "password": b"secret123",
            "api-key": b"abc-def-ghi"
        }
    )
    
    # Verify all files are created
    base_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-secret-multi"
    
    for filename, expected_content in [
        ("username", "admin"),
        ("password", "secret123"),
        ("api-key", "abc-def-ghi")
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


@pytest.mark.secret
def test_secret_ignored_without_labels(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test that Secrets without matching labels are ignored."""
    
    # Create Secret without matching labels
    secret = create_secret(
        k8s_client,
        name="test-secret-ignored",
        namespace=test_namespace,
        labels={"app": "different"},  # Different label value
        data={"ignored.txt": b"should not be synced"}
    )
    
    # Wait a bit to see if file appears (it shouldn't)
    time.sleep(5)
    
    # Verify file was NOT created
    file_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-secret-ignored/ignored.txt"
    
    exists = file_exists_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        file_path
    )
    
    assert not exists, f"File {file_path} should not have been created"


@pytest.mark.secret
def test_secret_tls_certificate(
    watcher_deployment: dict,
    k8s_client: client.CoreV1Api,
    test_namespace: str
):
    """Test syncing TLS certificate secrets."""
    
    # Create a TLS-type Secret
    tls_cert = b"-----BEGIN CERTIFICATE-----\nMOCK_CERT_DATA\n-----END CERTIFICATE-----"
    tls_key = b"-----BEGIN PRIVATE KEY-----\nMOCK_KEY_DATA\n-----END PRIVATE KEY-----"
    
    secret = create_secret(
        k8s_client,
        name="test-tls-secret",
        namespace=test_namespace,
        labels={"app": "test"},
        data={
            "tls.crt": tls_cert,
            "tls.key": tls_key
        },
        secret_type="kubernetes.io/tls"
    )
    
    # Verify both files are created
    base_path = f"/tmp/k8s-watcher-data/{test_namespace}/test-tls-secret"
    
    # Check certificate
    cert_path = f"{base_path}/tls.crt"
    assert wait_for_file_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        cert_path,
        timeout=30
    )
    
    cert_content = read_file_from_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        cert_path
    )
    assert "BEGIN CERTIFICATE" in cert_content
    
    # Check key
    key_path = f"{base_path}/tls.key"
    assert wait_for_file_in_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        key_path,
        timeout=30
    )
    
    key_content = read_file_from_pod(
        k8s_client,
        watcher_deployment["pod_name"],
        test_namespace,
        key_path
    )
    assert "BEGIN PRIVATE KEY" in key_content
