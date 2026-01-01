"""Helper utilities for k8s-watcher integration tests."""

import time
import subprocess
from typing import Optional, Dict, Any
from kubernetes import client
from kubernetes.stream import stream


def wait_for_pod_ready(
    v1: client.CoreV1Api,
    pod_name: str,
    namespace: str,
    timeout: int = 60
) -> bool:
    """
    Wait for a pod to be ready.
    
    Args:
        v1: Kubernetes CoreV1Api client
        pod_name: Name of the pod
        namespace: Namespace of the pod
        timeout: Maximum time to wait in seconds
        
    Returns:
        True if pod became ready, False otherwise
    """
    start_time = time.time()
    
    while time.time() - start_time < timeout:
        try:
            pod = v1.read_namespaced_pod(name=pod_name, namespace=namespace)
            
            if pod.status.phase == "Running":
                # Check if all containers are ready
                if pod.status.container_statuses:
                    all_ready = all(
                        container.ready
                        for container in pod.status.container_statuses
                    )
                    if all_ready:
                        return True
                        
        except client.exceptions.ApiException:
            pass
            
        time.sleep(1)
    
    return False


def wait_for_file_in_pod(
    v1: client.CoreV1Api,
    pod_name: str,
    namespace: str,
    file_path: str,
    timeout: int = 30
) -> bool:
    """
    Wait for a file to exist in a pod.
    
    Args:
        v1: Kubernetes CoreV1Api client
        pod_name: Name of the pod
        namespace: Namespace of the pod
        file_path: Path to the file inside the pod
        timeout: Maximum time to wait in seconds
        
    Returns:
        True if file exists, False otherwise
    """
    start_time = time.time()
    
    while time.time() - start_time < timeout:
        if file_exists_in_pod(v1, pod_name, namespace, file_path):
            return True
        time.sleep(1)
    
    print(f"Timed out waiting for file {file_path} in pod {pod_name}")
    logs = get_pod_logs(v1, pod_name, namespace)
    print(f"Pod logs:\n{logs}")
    return False


def file_exists_in_pod(
    v1: client.CoreV1Api,
    pod_name: str,
    namespace: str,
    file_path: str
) -> bool:
    """
    Check if a file exists in a pod.
    
    Args:
        v1: Kubernetes CoreV1Api client
        pod_name: Name of the pod
        namespace: Namespace of the pod
        file_path: Path to the file inside the pod
        
    Returns:
        True if file exists, False otherwise
    """
    try:
        exec_command = ['/bin/sh', '-c', f'test -f {file_path} && echo exists']
        resp = stream(
            v1.connect_get_namespaced_pod_exec,
            pod_name,
            namespace,
            command=exec_command,
            stderr=True,
            stdin=False,
            stdout=True,
            tty=False
        )
        return 'exists' in resp
    except Exception:
        return False


def read_file_from_pod(
    v1: client.CoreV1Api,
    pod_name: str,
    namespace: str,
    file_path: str
) -> Optional[str]:
    """
    Read file contents from a pod.
    
    Args:
        v1: Kubernetes CoreV1Api client
        pod_name: Name of the pod
        namespace: Namespace of the pod
        file_path: Path to the file inside the pod
        
    Returns:
        File contents as string, or None if file doesn't exist
    """
    try:
        exec_command = ['/bin/cat', file_path]
        resp = stream(
            v1.connect_get_namespaced_pod_exec,
            pod_name,
            namespace,
            command=exec_command,
            stderr=True,
            stdin=False,
            stdout=True,
            tty=False
        )
        return resp
    except Exception:
        return None


def create_configmap(
    v1: client.CoreV1Api,
    name: str,
    namespace: str,
    data: Dict[str, str],
    labels: Optional[Dict[str, str]] = None,
    annotations: Optional[Dict[str, str]] = None
) -> client.V1ConfigMap:
    """
    Create a ConfigMap.
    
    Args:
        v1: Kubernetes CoreV1Api client
        name: ConfigMap name
        namespace: Namespace
        data: ConfigMap data
        labels: Optional labels
        annotations: Optional annotations
        
    Returns:
        Created ConfigMap object
    """
    configmap = client.V1ConfigMap(
        metadata=client.V1ObjectMeta(
            name=name,
            labels=labels or {},
            annotations=annotations or {}
        ),
        data=data
    )
    
    return v1.create_namespaced_config_map(
        namespace=namespace,
        body=configmap
    )


def create_secret(
    v1: client.CoreV1Api,
    name: str,
    namespace: str,
    data: Dict[str, bytes],
    labels: Optional[Dict[str, str]] = None,
    annotations: Optional[Dict[str, str]] = None,
    secret_type: str = "Opaque"
) -> client.V1Secret:
    """
    Create a Secret.
    
    Args:
        v1: Kubernetes CoreV1Api client
        name: Secret name
        namespace: Namespace
        data: Secret data (as bytes)
        labels: Optional labels
        annotations: Optional annotations
        secret_type: Secret type (default: Opaque)
        
    Returns:
        Created Secret object
    """
    import base64
    
    # Encode data to base64
    encoded_data = {k: base64.b64encode(v).decode('utf-8') for k, v in data.items()}
    
    secret = client.V1Secret(
        metadata=client.V1ObjectMeta(
            name=name,
            labels=labels or {},
            annotations=annotations or {}
        ),
        type=secret_type,
        data=encoded_data
    )
    
    return v1.create_namespaced_secret(
        namespace=namespace,
        body=secret
    )


def delete_all_configmaps(v1: client.CoreV1Api, namespace: str):
    """Delete all ConfigMaps in a namespace."""
    try:
        v1.delete_collection_namespaced_config_map(namespace=namespace)
    except client.exceptions.ApiException:
        pass


def delete_all_secrets(v1: client.CoreV1Api, namespace: str):
    """Delete all Secrets in a namespace."""
    try:
        v1.delete_collection_namespaced_secret(namespace=namespace)
    except client.exceptions.ApiException:
        pass


def run_command(command: list, timeout: int = 30) -> tuple:
    """
    Run a shell command.
    
    Args:
        command: Command as list of strings
        timeout: Command timeout in seconds
        
    Returns:
        Tuple of (returncode, stdout, stderr)
    """
    try:
        result = subprocess.run(
            command,
            capture_output=True,
            text=True,
            timeout=timeout
        )
        return result.returncode, result.stdout, result.stderr
    except subprocess.TimeoutExpired:
        return -1, "", "Command timed out"


def get_pod_logs(
    v1: client.CoreV1Api,
    pod_name: str,
    namespace: str,
    tail_lines: int = 100
) -> str:
    """
    Get logs from a pod.
    
    Args:
        v1: Kubernetes CoreV1Api client
        pod_name: Name of the pod
        namespace: Namespace of the pod
        tail_lines: Number of lines to tail
        
    Returns:
        Pod logs as string
    """
    try:
        return v1.read_namespaced_pod_log(
            name=pod_name,
            namespace=namespace,
            tail_lines=tail_lines
        )
    except client.exceptions.ApiException:
        return ""
