"""Pytest fixtures for k8s-watcher integration tests."""

import os
import subprocess
import time
import pytest
from typing import Generator
from kubernetes import client, config
from pytest_httpserver import HTTPServer


@pytest.fixture(scope="session")
def kind_cluster() -> Generator[str, None, None]:
    """
    Create a KinD cluster for testing.
    
    Yields:
        Cluster name
    """
    cluster_name = "k8s-watcher-test"
    config_path = os.path.join(os.path.dirname(__file__), "kind-config.yaml")
    
    # Check if cluster already exists
    result = subprocess.run(
        ["kind", "get", "clusters"],
        capture_output=True,
        text=True
    )
    
    cluster_exists = cluster_name in result.stdout
    
    if not cluster_exists:
        print(f"\nCreating KinD cluster: {cluster_name}")
        subprocess.run(
            ["kind", "create", "cluster", "--config", config_path, "--name", cluster_name],
            check=True
        )
        # Wait for cluster to be ready
        time.sleep(5)
    else:
        print(f"\nUsing existing KinD cluster: {cluster_name}")
    
    yield cluster_name
    
    # Cleanup - delete cluster after all tests
    # Comment out to keep cluster for debugging
    print(f"\nDeleting KinD cluster: {cluster_name}")
    subprocess.run(["kind", "delete", "cluster", "--name", cluster_name])


@pytest.fixture(scope="session")
def docker_image(kind_cluster: str) -> Generator[str, None, None]:
    """
    Build and load the k8s-watcher Docker image into KinD.
    
    Args:
        kind_cluster: KinD cluster name
        
    Yields:
        Docker image name and tag
    """
    image_name = "k8s-watcher"
    image_tag = "test"
    full_image = f"{image_name}:{image_tag}"
    
    # Get the project root (two levels up from this file)
    project_root = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))
    
    print(f"\nBuilding Docker image: {full_image}")
    subprocess.run(
        ["docker", "build", "-t", full_image, "."],
        cwd=project_root,
        check=True
    )
    
    print(f"\nLoading image into KinD cluster: {kind_cluster}")
    subprocess.run(
        ["kind", "load", "docker-image", full_image, "--name", kind_cluster],
        check=True
    )
    
    yield full_image


@pytest.fixture(scope="session")
def k8s_client(kind_cluster: str) -> Generator[client.CoreV1Api, None, None]:
    """
    Create a Kubernetes API client for the KinD cluster.
    
    Args:
        kind_cluster: KinD cluster name
        
    Yields:
        Kubernetes CoreV1Api client
    """
    # Load kubeconfig for the KinD cluster
    kubeconfig_path = os.path.expanduser("~/.kube/config")
    config.load_kube_config(config_file=kubeconfig_path, context=f"kind-{kind_cluster}")
    
    yield client.CoreV1Api()


@pytest.fixture
def test_namespace(k8s_client: client.CoreV1Api) -> Generator[str, None, None]:
    """
    Create an isolated namespace for each test.
    
    Args:
        k8s_client: Kubernetes API client
        
    Yields:
        Namespace name
    """
    import uuid
    
    # Generate unique namespace name
    namespace_name = f"test-{uuid.uuid4().hex[:8]}"
    
    print(f"\nCreating namespace: {namespace_name}")
    
    # Create namespace
    namespace = client.V1Namespace(
        metadata=client.V1ObjectMeta(name=namespace_name)
    )
    k8s_client.create_namespace(body=namespace)
    
    # Wait a bit for namespace to be active
    time.sleep(1)
    
    yield namespace_name
    
    # Cleanup namespace
    print(f"\nDeleting namespace: {namespace_name}")
    try:
        k8s_client.delete_namespace(
            name=namespace_name,
            body=client.V1DeleteOptions()
        )
    except client.exceptions.ApiException:
        pass


@pytest.fixture
def watcher_config_basic() -> dict:
    """
    Basic watcher configuration for testing.
    
    Returns:
        Configuration dictionary
    """
    return {
        "output": {
            "folder": "/tmp/k8s-watcher-data",
            "folderAnnotation": "k8s-watcher-target-dir",
            "uniqueFilenames": False,
            "defaultFileMode": "0644"
        },
        "kubernetes": {
            "namespace": "default"
        },
        "resources": {
            "type": "both",
            "method": "WATCH",
            "watchConfig": {
                "serverTimeout": 60,
                "clientTimeout": 66,
                "errorThrottleTime": 5,
                "ignoreProcessed": True
            },
            "labels": [
                {
                    "name": "app",
                    "value": "test"
                }
            ]
        },
        "logging": {
            "level": "INFO",
            "format": "LOGFMT"
        }
    }


@pytest.fixture
def watcher_deployment(
    k8s_client: client.CoreV1Api,
    test_namespace: str,
    docker_image: str,
    watcher_config_basic: dict
) -> Generator[dict, None, None]:
    """
    Deploy k8s-watcher to the test namespace.
    
    Args:
        k8s_client: Kubernetes API client
        test_namespace: Test namespace
        docker_image: Docker image name
        watcher_config_basic: Basic watcher configuration
        
    Yields:
        Dictionary with deployment info (pod_name, namespace, etc.)
    """
    import yaml
    from helpers import wait_for_pod_ready
    
    # Update config to use test namespace
    watcher_config_basic["kubernetes"]["namespace"] = test_namespace
    
    # Create ConfigMap with watcher configuration
    config_cm = client.V1ConfigMap(
        metadata=client.V1ObjectMeta(name="watcher-config"),
        data={"config.yaml": yaml.dump(watcher_config_basic)}
    )
    k8s_client.create_namespaced_config_map(
        namespace=test_namespace,
        body=config_cm
    )
    
    # Create ServiceAccount
    rbac_v1 = client.RbacAuthorizationV1Api()
    sa = client.V1ServiceAccount(
        metadata=client.V1ObjectMeta(name="k8s-watcher")
    )
    k8s_client.create_namespaced_service_account(
        namespace=test_namespace,
        body=sa
    )
    
    # Create Role with permissions to read ConfigMaps and Secrets
    role = client.V1Role(
        metadata=client.V1ObjectMeta(name="k8s-watcher"),
        rules=[
            client.V1PolicyRule(
                api_groups=[""],
                resources=["configmaps", "secrets"],
                verbs=["get", "list", "watch"]
            )
        ]
    )
    rbac_v1.create_namespaced_role(
        namespace=test_namespace,
        body=role
    )
    
    # Create RoleBinding
    role_binding = client.V1RoleBinding(
        metadata=client.V1ObjectMeta(name="k8s-watcher"),
        role_ref=client.V1RoleRef(
            api_group="rbac.authorization.k8s.io",
            kind="Role",
            name="k8s-watcher"
        ),
        subjects=[
            client.RbacV1Subject(
                kind="ServiceAccount",
                name="k8s-watcher",
                namespace=test_namespace
            )
        ]
    )
    rbac_v1.create_namespaced_role_binding(
        namespace=test_namespace,
        body=role_binding
    )
    
    # Create Deployment
    apps_v1 = client.AppsV1Api()
    deployment = client.V1Deployment(
        metadata=client.V1ObjectMeta(name="k8s-watcher"),
        spec=client.V1DeploymentSpec(
            replicas=1,
            selector=client.V1LabelSelector(
                match_labels={"app": "k8s-watcher"}
            ),
            template=client.V1PodTemplateSpec(
                metadata=client.V1ObjectMeta(
                    labels={"app": "k8s-watcher"}
                ),
                spec=client.V1PodSpec(
                    service_account_name="k8s-watcher",
                    containers=[
                        client.V1Container(
                            name="watcher",
                            image=docker_image,
                            image_pull_policy="Never",  # Use local image
                            args=["-config", "/etc/k8s-watcher/config.yaml"],
                            volume_mounts=[
                                client.V1VolumeMount(
                                    name="config",
                                    mount_path="/etc/k8s-watcher"
                                ),
                                client.V1VolumeMount(
                                    name="data",
                                    mount_path="/tmp/k8s-watcher-data"
                                )
                            ]
                        )
                    ],
                    volumes=[
                        client.V1Volume(
                            name="config",
                            config_map=client.V1ConfigMapVolumeSource(
                                name="watcher-config"
                            )
                        ),
                        client.V1Volume(
                            name="data",
                            empty_dir=client.V1EmptyDirVolumeSource()
                        )
                    ]
                )
            )
        )
    )
    
    print(f"\nDeploying k8s-watcher to namespace: {test_namespace}")
    apps_v1.create_namespaced_deployment(
        namespace=test_namespace,
        body=deployment
    )
    
    # Wait for pod to be ready
    time.sleep(2)
    
    # Get pod name
    pods = k8s_client.list_namespaced_pod(
        namespace=test_namespace,
        label_selector="app=k8s-watcher"
    )
    
    if not pods.items:
        raise RuntimeError("k8s-watcher pod not found")
    
    pod_name = pods.items[0].metadata.name
    
    print(f"Waiting for pod {pod_name} to be ready...")
    if not wait_for_pod_ready(k8s_client, pod_name, test_namespace, timeout=60):
        # Get logs for debugging
        from helpers import get_pod_logs
        logs = get_pod_logs(k8s_client, pod_name, test_namespace)
        print(f"Pod logs:\n{logs}")
        raise RuntimeError(f"Pod {pod_name} did not become ready in time")
    
    print(f"Pod {pod_name} is ready")
    
    deployment_info = {
        "pod_name": pod_name,
        "namespace": test_namespace,
        "deployment_name": "k8s-watcher"
    }
    
    yield deployment_info
    
    # Cleanup handled by namespace deletion


@pytest.fixture
def webhook_server(httpserver: HTTPServer) -> HTTPServer:
    """
    Mock webhook server for testing webhook notifications.
    
    Args:
        httpserver: pytest-httpserver fixture
        
    Returns:
        HTTPServer instance
    """
    # Configure default response for webhook endpoint
    httpserver.expect_request(
        "/webhook",
        method="POST"
    ).respond_with_json(
        {"status": "ok"},
        status=200
    )
    
    return httpserver


@pytest.fixture
def mock_webhook_server(
    k8s_client: client.CoreV1Api,
    test_namespace: str
) -> Generator[dict, None, None]:
    """
    Deploy Mockolate mock HTTP server in the test namespace.
    
    Args:
        k8s_client: Kubernetes API client
        test_namespace: Test namespace
        
    Yields:
        Dictionary with mock server info (service_name, url)
    """
    from helpers import wait_for_pod_ready
    
    # Create ConfigMap with Mockolate config
    mockolate_config = """
endpoints:
  /webhook:
    - method: POST
      content: application/json
      body: '{"status": "received"}'
      status: 200
  /webhook/auth:
    - method: POST
      content: application/json
      body: '{"status": "authenticated"}'
      status: 200
"""
    config_cm = client.V1ConfigMap(
        metadata=client.V1ObjectMeta(name="mockolate-config"),
        data={"server.yaml": mockolate_config}
    )
    k8s_client.create_namespaced_config_map(
        namespace=test_namespace,
        body=config_cm
    )
    
    # Create Mockolate Pod
    pod = client.V1Pod(
        metadata=client.V1ObjectMeta(
            name="mockolate",
            labels={"app": "mockolate"}
        ),
        spec=client.V1PodSpec(
            containers=[
                client.V1Container(
                    name="mockolate",
                    image="nihalwasim/mock-http-server:latest",
                    image_pull_policy="IfNotPresent",
                    ports=[
                        client.V1ContainerPort(container_port=8080)
                    ],
                    command=["/mock-server"],
                    args=["--server.config=/etc/config/server.yaml"],
                    volume_mounts=[
                        client.V1VolumeMount(
                            name="config",
                            mount_path="/etc/config",
                            read_only=True
                        )
                    ]
                )
            ],
            volumes=[
                client.V1Volume(
                    name="config",
                    config_map=client.V1ConfigMapVolumeSource(
                        name="mockolate-config"
                    )
                )
            ]
        )
    )
    
    print(f"\nDeploying Mockolate mock server to namespace: {test_namespace}")
    k8s_client.create_namespaced_pod(
        namespace=test_namespace,
        body=pod
    )
    
    # Create Service for Mockolate
    service = client.V1Service(
        metadata=client.V1ObjectMeta(name="mockolate"),
        spec=client.V1ServiceSpec(
            selector={"app": "mockolate"},
            ports=[
                client.V1ServicePort(
                    protocol="TCP",
                    port=8080,
                    target_port=8080
                )
            ]
        )
    )
    k8s_client.create_namespaced_service(
        namespace=test_namespace,
        body=service
    )
    
    # Wait for pod to be ready
    print("Waiting for Mockolate pod to be ready...")
    if not wait_for_pod_ready(k8s_client, "mockolate", test_namespace, timeout=60):
        from helpers import get_pod_logs
        logs = get_pod_logs(k8s_client, "mockolate", test_namespace)
        print(f"Mockolate logs:\n{logs}")
        raise RuntimeError("Mockolate pod did not become ready in time")
    
    print("Mockolate mock server is ready")
    
    # The service URL is: http://mockolate.<namespace>.svc.cluster.local:8080
    service_url = f"http://mockolate.{test_namespace}.svc.cluster.local:8080"
    
    yield {
        "service_name": "mockolate",
        "url": service_url,
        "webhook_endpoint": f"{service_url}/webhook"
    }
    
    # Cleanup handled by namespace deletion


@pytest.fixture
def watcher_config_webhook(mock_webhook_server: dict) -> dict:
    """
    Watcher configuration with webhook enabled.
    
    Args:
        mock_webhook_server: Mock webhook server info
        
    Returns:
        Configuration dictionary with webhook
    """
    return {
        "output": {
            "folder": "/tmp/k8s-watcher-data",
            "folderAnnotation": "k8s-watcher-target-dir",
            "uniqueFilenames": False,
            "defaultFileMode": "0644"
        },
        "kubernetes": {
            "namespace": "default"
        },
        "resources": {
            "type": "both",
            "method": "WATCH",
            "watchConfig": {
                "serverTimeout": 60,
                "clientTimeout": 66,
                "errorThrottleTime": 5,
                "ignoreProcessed": True
            },
            "labels": [
                {
                    "name": "app",
                    "value": "webhook-test",
                    "request": {
                        "url": mock_webhook_server["webhook_endpoint"],
                        "method": "POST",
                        "timeout": 10,
                        "retry": {
                            "total": 3,
                            "backoffFactor": 1.5
                        }
                    }
                }
            ]
        },
        "logging": {
            "level": "DEBUG",
            "format": "LOGFMT"
        }
    }


@pytest.fixture
def webhook_watcher_deployment(
    k8s_client: client.CoreV1Api,
    test_namespace: str,
    docker_image: str,
    watcher_config_webhook: dict
) -> Generator[dict, None, None]:
    """
    Deploy k8s-watcher with webhook configuration.
    
    Args:
        k8s_client: Kubernetes API client
        test_namespace: Test namespace
        docker_image: Docker image name
        watcher_config_webhook: Watcher config with webhook
        
    Yields:
        Dictionary with deployment info
    """
    import yaml
    from helpers import wait_for_pod_ready
    
    # Update config to use test namespace
    watcher_config_webhook["kubernetes"]["namespace"] = test_namespace
    
    # Create ConfigMap with watcher configuration
    config_cm = client.V1ConfigMap(
        metadata=client.V1ObjectMeta(name="watcher-config"),
        data={"config.yaml": yaml.dump(watcher_config_webhook)}
    )
    k8s_client.create_namespaced_config_map(
        namespace=test_namespace,
        body=config_cm
    )
    
    # Create ServiceAccount
    rbac_v1 = client.RbacAuthorizationV1Api()
    sa = client.V1ServiceAccount(
        metadata=client.V1ObjectMeta(name="k8s-watcher")
    )
    k8s_client.create_namespaced_service_account(
        namespace=test_namespace,
        body=sa
    )
    
    # Create Role
    role = client.V1Role(
        metadata=client.V1ObjectMeta(name="k8s-watcher"),
        rules=[
            client.V1PolicyRule(
                api_groups=[""],
                resources=["configmaps", "secrets"],
                verbs=["get", "list", "watch"]
            )
        ]
    )
    rbac_v1.create_namespaced_role(
        namespace=test_namespace,
        body=role
    )
    
    # Create RoleBinding
    role_binding = client.V1RoleBinding(
        metadata=client.V1ObjectMeta(name="k8s-watcher"),
        role_ref=client.V1RoleRef(
            api_group="rbac.authorization.k8s.io",
            kind="Role",
            name="k8s-watcher"
        ),
        subjects=[
            client.RbacV1Subject(
                kind="ServiceAccount",
                name="k8s-watcher",
                namespace=test_namespace
            )
        ]
    )
    rbac_v1.create_namespaced_role_binding(
        namespace=test_namespace,
        body=role_binding
    )
    
    # Create Deployment
    apps_v1 = client.AppsV1Api()
    deployment = client.V1Deployment(
        metadata=client.V1ObjectMeta(name="k8s-watcher"),
        spec=client.V1DeploymentSpec(
            replicas=1,
            selector=client.V1LabelSelector(
                match_labels={"app": "k8s-watcher"}
            ),
            template=client.V1PodTemplateSpec(
                metadata=client.V1ObjectMeta(
                    labels={"app": "k8s-watcher"}
                ),
                spec=client.V1PodSpec(
                    service_account_name="k8s-watcher",
                    containers=[
                        client.V1Container(
                            name="watcher",
                            image=docker_image,
                            image_pull_policy="Never",
                            args=["-config", "/etc/k8s-watcher/config.yaml"],
                            volume_mounts=[
                                client.V1VolumeMount(
                                    name="config",
                                    mount_path="/etc/k8s-watcher"
                                ),
                                client.V1VolumeMount(
                                    name="data",
                                    mount_path="/tmp/k8s-watcher-data"
                                )
                            ]
                        )
                    ],
                    volumes=[
                        client.V1Volume(
                            name="config",
                            config_map=client.V1ConfigMapVolumeSource(
                                name="watcher-config"
                            )
                        ),
                        client.V1Volume(
                            name="data",
                            empty_dir=client.V1EmptyDirVolumeSource()
                        )
                    ]
                )
            )
        )
    )
    
    print(f"\nDeploying k8s-watcher (with webhook) to namespace: {test_namespace}")
    apps_v1.create_namespaced_deployment(
        namespace=test_namespace,
        body=deployment
    )
    
    # Wait for pod to be ready
    time.sleep(2)
    
    pods = k8s_client.list_namespaced_pod(
        namespace=test_namespace,
        label_selector="app=k8s-watcher"
    )
    
    if not pods.items:
        raise RuntimeError("k8s-watcher pod not found")
    
    pod_name = pods.items[0].metadata.name
    
    print(f"Waiting for pod {pod_name} to be ready...")
    if not wait_for_pod_ready(k8s_client, pod_name, test_namespace, timeout=60):
        from helpers import get_pod_logs
        logs = get_pod_logs(k8s_client, pod_name, test_namespace)
        print(f"Pod logs:\n{logs}")
        raise RuntimeError(f"Pod {pod_name} did not become ready in time")
    
    print(f"Pod {pod_name} is ready")
    
    yield {
        "pod_name": pod_name,
        "namespace": test_namespace,
        "deployment_name": "k8s-watcher"
    }
    
    # Cleanup handled by namespace deletion


@pytest.fixture
def mock_webhook_server_auth(
    k8s_client: client.CoreV1Api,
    test_namespace: str
) -> Generator[dict, None, None]:
    """
    Deploy Mockolate with basic auth enabled.
    
    Args:
        k8s_client: Kubernetes API client
        test_namespace: Test namespace
        
    Yields:
        Dictionary with mock server info including auth credentials
    """
    from helpers import wait_for_pod_ready
    
    # Auth credentials
    auth_username = "testuser"
    auth_password = "testpass123"
    
    # Create ConfigMap with Mockolate config
    mockolate_config = """
endpoints:
  /webhook/auth:
    - method: POST
      content: application/json
      body: '{"status": "authenticated"}'
      status: 200
"""
    config_cm = client.V1ConfigMap(
        metadata=client.V1ObjectMeta(name="mockolate-auth-config"),
        data={"server.yaml": mockolate_config}
    )
    k8s_client.create_namespaced_config_map(
        namespace=test_namespace,
        body=config_cm
    )
    
    # Create Mockolate Pod with basic auth enabled
    pod = client.V1Pod(
        metadata=client.V1ObjectMeta(
            name="mockolate-auth",
            labels={"app": "mockolate-auth"}
        ),
        spec=client.V1PodSpec(
            containers=[
                client.V1Container(
                    name="mockolate",
                    image="nihalwasim/mock-http-server:latest",
                    image_pull_policy="IfNotPresent",
                    ports=[
                        client.V1ContainerPort(container_port=8080)
                    ],
                    command=["/mock-server"],
                    args=[
                        "--server.config=/etc/config/server.yaml",
                        f"--basicauth.username={auth_username}",
                        f"--basicauth.password={auth_password}"
                    ],
                    volume_mounts=[
                        client.V1VolumeMount(
                            name="config",
                            mount_path="/etc/config",
                            read_only=True
                        )
                    ]
                )
            ],
            volumes=[
                client.V1Volume(
                    name="config",
                    config_map=client.V1ConfigMapVolumeSource(
                        name="mockolate-auth-config"
                    )
                )
            ]
        )
    )
    
    print(f"\n🔐 Deploying Mockolate (with auth) to namespace: {test_namespace}")
    k8s_client.create_namespaced_pod(
        namespace=test_namespace,
        body=pod
    )
    
    # Create Service
    service = client.V1Service(
        metadata=client.V1ObjectMeta(name="mockolate-auth"),
        spec=client.V1ServiceSpec(
            selector={"app": "mockolate-auth"},
            ports=[
                client.V1ServicePort(
                    protocol="TCP",
                    port=8080,
                    target_port=8080
                )
            ]
        )
    )
    k8s_client.create_namespaced_service(
        namespace=test_namespace,
        body=service
    )
    
    # Wait for pod to be ready
    print("⏳ Waiting for Mockolate (auth) pod to be ready...")
    if not wait_for_pod_ready(k8s_client, "mockolate-auth", test_namespace, timeout=60):
        from helpers import get_pod_logs
        logs = get_pod_logs(k8s_client, "mockolate-auth", test_namespace)
        print(f"Mockolate logs:\n{logs}")
        raise RuntimeError("Mockolate (auth) pod did not become ready in time")
    
    print("✓ Mockolate (auth) mock server is ready")
    
    service_url = f"http://mockolate-auth.{test_namespace}.svc.cluster.local:8080"
    
    yield {
        "service_name": "mockolate-auth",
        "url": service_url,
        "webhook_endpoint": f"{service_url}/webhook/auth",
        "username": auth_username,
        "password": auth_password
    }


@pytest.fixture
def watcher_config_webhook_auth(mock_webhook_server_auth: dict) -> dict:
    """
    Watcher configuration with webhook and basic auth.
    """
    return {
        "output": {
            "folder": "/tmp/k8s-watcher-data",
            "folderAnnotation": "k8s-watcher-target-dir",
            "uniqueFilenames": False,
            "defaultFileMode": "0644"
        },
        "kubernetes": {
            "namespace": "default"
        },
        "resources": {
            "type": "both",
            "method": "WATCH",
            "watchConfig": {
                "serverTimeout": 60,
                "clientTimeout": 66,
                "errorThrottleTime": 5,
                "ignoreProcessed": True
            },
            "labels": [
                {
                    "name": "app",
                    "value": "webhook-auth-test",
                    "request": {
                        "url": mock_webhook_server_auth["webhook_endpoint"],
                        "method": "POST",
                        "timeout": 10,
                        "auth": {
                            "basic": {
                                "username": mock_webhook_server_auth["username"],
                                "password": mock_webhook_server_auth["password"]
                            }
                        },
                        "retry": {
                            "total": 3,
                            "backoffFactor": 1.5
                        }
                    }
                }
            ]
        },
        "logging": {
            "level": "DEBUG",
            "format": "LOGFMT"
        }
    }


@pytest.fixture
def webhook_watcher_deployment_auth(
    k8s_client: client.CoreV1Api,
    test_namespace: str,
    docker_image: str,
    watcher_config_webhook_auth: dict
) -> Generator[dict, None, None]:
    """
    Deploy k8s-watcher with webhook + basic auth configuration.
    """
    import yaml
    from helpers import wait_for_pod_ready
    
    watcher_config_webhook_auth["kubernetes"]["namespace"] = test_namespace
    
    config_cm = client.V1ConfigMap(
        metadata=client.V1ObjectMeta(name="watcher-config"),
        data={"config.yaml": yaml.dump(watcher_config_webhook_auth)}
    )
    k8s_client.create_namespaced_config_map(
        namespace=test_namespace,
        body=config_cm
    )
    
    # Create RBAC resources
    rbac_v1 = client.RbacAuthorizationV1Api()
    sa = client.V1ServiceAccount(
        metadata=client.V1ObjectMeta(name="k8s-watcher")
    )
    k8s_client.create_namespaced_service_account(
        namespace=test_namespace,
        body=sa
    )
    
    role = client.V1Role(
        metadata=client.V1ObjectMeta(name="k8s-watcher"),
        rules=[
            client.V1PolicyRule(
                api_groups=[""],
                resources=["configmaps", "secrets"],
                verbs=["get", "list", "watch"]
            )
        ]
    )
    rbac_v1.create_namespaced_role(
        namespace=test_namespace,
        body=role
    )
    
    role_binding = client.V1RoleBinding(
        metadata=client.V1ObjectMeta(name="k8s-watcher"),
        role_ref=client.V1RoleRef(
            api_group="rbac.authorization.k8s.io",
            kind="Role",
            name="k8s-watcher"
        ),
        subjects=[
            client.RbacV1Subject(
                kind="ServiceAccount",
                name="k8s-watcher",
                namespace=test_namespace
            )
        ]
    )
    rbac_v1.create_namespaced_role_binding(
        namespace=test_namespace,
        body=role_binding
    )
    
    # Create Deployment
    apps_v1 = client.AppsV1Api()
    deployment = client.V1Deployment(
        metadata=client.V1ObjectMeta(name="k8s-watcher"),
        spec=client.V1DeploymentSpec(
            replicas=1,
            selector=client.V1LabelSelector(
                match_labels={"app": "k8s-watcher"}
            ),
            template=client.V1PodTemplateSpec(
                metadata=client.V1ObjectMeta(
                    labels={"app": "k8s-watcher"}
                ),
                spec=client.V1PodSpec(
                    service_account_name="k8s-watcher",
                    containers=[
                        client.V1Container(
                            name="watcher",
                            image=docker_image,
                            image_pull_policy="Never",
                            args=["-config", "/etc/k8s-watcher/config.yaml"],
                            volume_mounts=[
                                client.V1VolumeMount(
                                    name="config",
                                    mount_path="/etc/k8s-watcher"
                                ),
                                client.V1VolumeMount(
                                    name="data",
                                    mount_path="/tmp/k8s-watcher-data"
                                )
                            ]
                        )
                    ],
                    volumes=[
                        client.V1Volume(
                            name="config",
                            config_map=client.V1ConfigMapVolumeSource(
                                name="watcher-config"
                            )
                        ),
                        client.V1Volume(
                            name="data",
                            empty_dir=client.V1EmptyDirVolumeSource()
                        )
                    ]
                )
            )
        )
    )
    
    print(f"\n🚀 Deploying k8s-watcher (with auth) to namespace: {test_namespace}")
    apps_v1.create_namespaced_deployment(
        namespace=test_namespace,
        body=deployment
    )
    
    time.sleep(2)
    
    pods = k8s_client.list_namespaced_pod(
        namespace=test_namespace,
        label_selector="app=k8s-watcher"
    )
    
    if not pods.items:
        raise RuntimeError("k8s-watcher pod not found")
    
    pod_name = pods.items[0].metadata.name
    
    print(f"⏳ Waiting for pod {pod_name} to be ready...")
    if not wait_for_pod_ready(k8s_client, pod_name, test_namespace, timeout=60):
        from helpers import get_pod_logs
        logs = get_pod_logs(k8s_client, pod_name, test_namespace)
        print(f"Pod logs:\n{logs}")
        raise RuntimeError(f"Pod {pod_name} did not become ready in time")
    
    print(f"✓ Pod {pod_name} is ready")
    
    yield {
        "pod_name": pod_name,
        "namespace": test_namespace,
        "deployment_name": "k8s-watcher"
    }


@pytest.fixture(scope="session")
def cert_manager_installed(kind_cluster: str) -> Generator[bool, None, None]:
    """
    Install cert-manager in the KinD cluster.
    """
    import subprocess
    
    # Check if cert-manager is already installed
    result = subprocess.run(
        ["kubectl", "get", "namespace", "cert-manager"],
        capture_output=True,
        text=True
    )
    
    if result.returncode != 0:
        print("\n📜 Installing cert-manager...")
        subprocess.run(
            ["kubectl", "apply", "-f", 
             "https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml"],
            check=True
        )
        
        # Wait for cert-manager to be ready
        print("⏳ Waiting for cert-manager to be ready...")
        time.sleep(30)  # Give cert-manager time to start
        
        subprocess.run(
            ["kubectl", "wait", "--for=condition=Available", 
             "deployment/cert-manager-webhook", "-n", "cert-manager", 
             "--timeout=120s"],
            check=True
        )
        print("✓ cert-manager installed and ready")
    else:
        print("\n✓ cert-manager already installed")
    
    
    yield True
    
    # Cleanup cert-manager
    print("\n🗑️  Uninstalling cert-manager...")
    subprocess.run(
        ["kubectl", "delete", "-f", 
         "https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml"],
        check=False  # Don't fail if already deleted or cluster is gone
    )
    print("✓ cert-manager uninstalled")


@pytest.fixture
def mock_webhook_server_tls(
    k8s_client: client.CoreV1Api,
    test_namespace: str,
    cert_manager_installed: bool
) -> Generator[dict, None, None]:
    """
    Deploy Mockolate with TLS using cert-manager.
    """
    from helpers import wait_for_pod_ready
    
    # Create a self-signed Issuer
    custom_api = client.CustomObjectsApi()
    
    issuer = {
        "apiVersion": "cert-manager.io/v1",
        "kind": "Issuer",
        "metadata": {
            "name": "selfsigned-issuer",
            "namespace": test_namespace
        },
        "spec": {
            "selfSigned": {}
        }
    }
    
    custom_api.create_namespaced_custom_object(
        group="cert-manager.io",
        version="v1",
        namespace=test_namespace,
        plural="issuers",
        body=issuer
    )
    
    # Create Certificate
    certificate = {
        "apiVersion": "cert-manager.io/v1",
        "kind": "Certificate",
        "metadata": {
            "name": "mockolate-tls",
            "namespace": test_namespace
        },
        "spec": {
            "secretName": "mockolate-tls-secret",
            "duration": "2160h",  # 90d
            "renewBefore": "360h",  # 15d
            "issuerRef": {
                "name": "selfsigned-issuer",
                "kind": "Issuer"
            },
            "dnsNames": [
                f"mockolate-tls.{test_namespace}.svc.cluster.local",
                "mockolate-tls"
            ]
        }
    }
    
    custom_api.create_namespaced_custom_object(
        group="cert-manager.io",
        version="v1",
        namespace=test_namespace,
        plural="certificates",
        body=certificate
    )
    
    # Wait for certificate to be ready
    print("⏳ Waiting for TLS certificate to be ready...")
    time.sleep(10)
    
    # Create ConfigMap with Mockolate config
    mockolate_config = """
endpoints:
  /webhook/tls:
    - method: POST
      content: application/json
      body: '{"status": "tls-success"}'
      status: 200
"""
    config_cm = client.V1ConfigMap(
        metadata=client.V1ObjectMeta(name="mockolate-tls-config"),
        data={"server.yaml": mockolate_config}
    )
    k8s_client.create_namespaced_config_map(
        namespace=test_namespace,
        body=config_cm
    )
    
    # Create Mockolate Pod with TLS
    pod = client.V1Pod(
        metadata=client.V1ObjectMeta(
            name="mockolate-tls",
            labels={"app": "mockolate-tls"}
        ),
        spec=client.V1PodSpec(
            containers=[
                client.V1Container(
                    name="mockolate",
                    image="nihalwasim/mock-http-server:latest",
                    image_pull_policy="IfNotPresent",
                    ports=[
                        client.V1ContainerPort(container_port=8443)
                    ],
                    command=["/mock-server"],
                    args=[
                        "--server.config=/etc/config/server.yaml",
                        "--port=8443",
                        "--tls",
                        "--tlsCertFile=/etc/tls/tls.crt",
                        "--tlsKeyFile=/etc/tls/tls.key"
                    ],
                    volume_mounts=[
                        client.V1VolumeMount(
                            name="config",
                            mount_path="/etc/config",
                            read_only=True
                        ),
                        client.V1VolumeMount(
                            name="tls",
                            mount_path="/etc/tls",
                            read_only=True
                        )
                    ]
                )
            ],
            volumes=[
                client.V1Volume(
                    name="config",
                    config_map=client.V1ConfigMapVolumeSource(
                        name="mockolate-tls-config"
                    )
                ),
                client.V1Volume(
                    name="tls",
                    secret=client.V1SecretVolumeSource(
                        secret_name="mockolate-tls-secret"
                    )
                )
            ]
        )
    )
    
    print(f"\n🔒 Deploying Mockolate (TLS) to namespace: {test_namespace}")
    k8s_client.create_namespaced_pod(
        namespace=test_namespace,
        body=pod
    )
    
    # Create Service
    service = client.V1Service(
        metadata=client.V1ObjectMeta(name="mockolate-tls"),
        spec=client.V1ServiceSpec(
            selector={"app": "mockolate-tls"},
            ports=[
                client.V1ServicePort(
                    protocol="TCP",
                    port=8443,
                    target_port=8443
                )
            ]
        )
    )
    k8s_client.create_namespaced_service(
        namespace=test_namespace,
        body=service
    )
    
    # Wait for pod
    print("⏳ Waiting for Mockolate (TLS) pod to be ready...")
    if not wait_for_pod_ready(k8s_client, "mockolate-tls", test_namespace, timeout=60):
        from helpers import get_pod_logs
        logs = get_pod_logs(k8s_client, "mockolate-tls", test_namespace)
        print(f"Mockolate logs:\n{logs}")
        raise RuntimeError("Mockolate (TLS) pod did not become ready in time")
    
    print("✓ Mockolate (TLS) mock server is ready")
    
    service_url = f"https://mockolate-tls.{test_namespace}.svc.cluster.local:8443"
    
    yield {
        "service_name": "mockolate-tls",
        "url": service_url,
        "webhook_endpoint": f"{service_url}/webhook/tls"
    }


@pytest.fixture
def watcher_config_webhook_tls(mock_webhook_server_tls: dict) -> dict:
    """
    Watcher configuration with HTTPS webhook (skipTLSVerify: true for self-signed).
    """
    return {
        "output": {
            "folder": "/tmp/k8s-watcher-data",
            "folderAnnotation": "k8s-watcher-target-dir",
            "uniqueFilenames": False,
            "defaultFileMode": "0644"
        },
        "kubernetes": {
            "namespace": "default"
        },
        "resources": {
            "type": "both",
            "method": "WATCH",
            "watchConfig": {
                "serverTimeout": 60,
                "clientTimeout": 66,
                "errorThrottleTime": 5,
                "ignoreProcessed": True
            },
            "labels": [
                {
                    "name": "app",
                    "value": "webhook-tls-test",
                    "request": {
                        "url": mock_webhook_server_tls["webhook_endpoint"],
                        "method": "POST",
                        "timeout": 10,
                        "skipTLSVerify": True,  # Required for self-signed certs
                        "retry": {
                            "total": 3,
                            "backoffFactor": 1.5
                        }
                    }
                }
            ]
        },
        "logging": {
            "level": "DEBUG",
            "format": "LOGFMT"
        }
    }


@pytest.fixture
def webhook_watcher_deployment_tls(
    k8s_client: client.CoreV1Api,
    test_namespace: str,
    docker_image: str,
    watcher_config_webhook_tls: dict
) -> Generator[dict, None, None]:
    """
    Deploy k8s-watcher with TLS webhook configuration.
    """
    import yaml
    from helpers import wait_for_pod_ready
    
    watcher_config_webhook_tls["kubernetes"]["namespace"] = test_namespace
    
    config_cm = client.V1ConfigMap(
        metadata=client.V1ObjectMeta(name="watcher-config"),
        data={"config.yaml": yaml.dump(watcher_config_webhook_tls)}
    )
    k8s_client.create_namespaced_config_map(
        namespace=test_namespace,
        body=config_cm
    )
    
    # Create RBAC resources
    rbac_v1 = client.RbacAuthorizationV1Api()
    sa = client.V1ServiceAccount(
        metadata=client.V1ObjectMeta(name="k8s-watcher")
    )
    k8s_client.create_namespaced_service_account(
        namespace=test_namespace,
        body=sa
    )
    
    role = client.V1Role(
        metadata=client.V1ObjectMeta(name="k8s-watcher"),
        rules=[
            client.V1PolicyRule(
                api_groups=[""],
                resources=["configmaps", "secrets"],
                verbs=["get", "list", "watch"]
            )
        ]
    )
    rbac_v1.create_namespaced_role(
        namespace=test_namespace,
        body=role
    )
    
    role_binding = client.V1RoleBinding(
        metadata=client.V1ObjectMeta(name="k8s-watcher"),
        role_ref=client.V1RoleRef(
            api_group="rbac.authorization.k8s.io",
            kind="Role",
            name="k8s-watcher"
        ),
        subjects=[
            client.RbacV1Subject(
                kind="ServiceAccount",
                name="k8s-watcher",
                namespace=test_namespace
            )
        ]
    )
    rbac_v1.create_namespaced_role_binding(
        namespace=test_namespace,
        body=role_binding
    )
    
    # Create Deployment
    apps_v1 = client.AppsV1Api()
    deployment = client.V1Deployment(
        metadata=client.V1ObjectMeta(name="k8s-watcher"),
        spec=client.V1DeploymentSpec(
            replicas=1,
            selector=client.V1LabelSelector(
                match_labels={"app": "k8s-watcher"}
            ),
            template=client.V1PodTemplateSpec(
                metadata=client.V1ObjectMeta(
                    labels={"app": "k8s-watcher"}
                ),
                spec=client.V1PodSpec(
                    service_account_name="k8s-watcher",
                    containers=[
                        client.V1Container(
                            name="watcher",
                            image=docker_image,
                            image_pull_policy="Never",
                            args=["-config", "/etc/k8s-watcher/config.yaml"],
                            volume_mounts=[
                                client.V1VolumeMount(
                                    name="config",
                                    mount_path="/etc/k8s-watcher"
                                ),
                                client.V1VolumeMount(
                                    name="data",
                                    mount_path="/tmp/k8s-watcher-data"
                                )
                            ]
                        )
                    ],
                    volumes=[
                        client.V1Volume(
                            name="config",
                            config_map=client.V1ConfigMapVolumeSource(
                                name="watcher-config"
                            )
                        ),
                        client.V1Volume(
                            name="data",
                            empty_dir=client.V1EmptyDirVolumeSource()
                        )
                    ]
                )
            )
        )
    )
    
    print(f"\n🚀 Deploying k8s-watcher (TLS) to namespace: {test_namespace}")
    apps_v1.create_namespaced_deployment(
        namespace=test_namespace,
        body=deployment
    )
    
    time.sleep(2)
    
    pods = k8s_client.list_namespaced_pod(
        namespace=test_namespace,
        label_selector="app=k8s-watcher"
    )
    
    if not pods.items:
        raise RuntimeError("k8s-watcher pod not found")
    
    pod_name = pods.items[0].metadata.name
    
    print(f"⏳ Waiting for pod {pod_name} to be ready...")
    if not wait_for_pod_ready(k8s_client, pod_name, test_namespace, timeout=60):
        from helpers import get_pod_logs
        logs = get_pod_logs(k8s_client, pod_name, test_namespace)
        print(f"Pod logs:\n{logs}")
        raise RuntimeError(f"Pod {pod_name} did not become ready in time")
    
    print(f"✓ Pod {pod_name} is ready")
    
    yield {
        "pod_name": pod_name,
        "namespace": test_namespace,
        "deployment_name": "k8s-watcher"
    }
