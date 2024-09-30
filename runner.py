import subprocess
import os
import argparse
from typing import Optional, Dict


class Resource:
    def __init__(self, resource_name: str, path: str, env: Optional[Dict[str, str]] = None):
        self.resource_name = resource_name
        self.path = path
        self.env = env


def create_docker_network() -> subprocess.CompletedProcess:
    return subprocess.run(["docker", "network", "create", "otel-net"], check=True)


def remove_docker_network() -> subprocess.CompletedProcess:
    return subprocess.run(["docker", "network", "rm", "otel-net"], check=True)


def up_docker_compose(resource: Resource) -> subprocess.CompletedProcess:
    if resource.env:
        os.environ.update(resource.env)

    return subprocess.run(
        ["docker", "compose", "-f", "docker-compose.yml", "up", "-d"],
        cwd=resource.path,
        check=True
    )


def down_docker_compose(resource: Resource) -> subprocess.CompletedProcess:
    return subprocess.run(
        ["docker", "compose", "-f", "docker-compose.yml", "down"],
        cwd=resource.path,
        check=True
    )


def run_jar_with_javaagent() -> subprocess.CompletedProcess:
    jar_name = "spring-petclinic-3.3.0-SNAPSHOT.jar"

    resource = Resource(
        resource_name="java-application",
        path="./java-application",
        env={
            "OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:9987",
            "OTEL_EXPORTER_OTLP_PROTOCOL": "grpc",
            "OTEL_TRACES_EXPORTER": "otlp",
            "OTEL_METRICS_EXPORTER": "otlp",
            "OTEL_LOGS_EXPORTER": "otlp"
        }
    )

    if resource.env:
        os.environ.update(resource.env)

    return subprocess.run(
        ["java", "-javaagent:opentelemetry-javaagent.jar", "-jar", jar_name],
        cwd=resource.path,
        check=True
    )


def main():
    parser = argparse.ArgumentParser(description="otel docker compose manager")
    parser.add_argument("command", choices=["up", "down", "jar"], help="up / down / jar")
    args = parser.parse_args()

    resources = [
        Resource("jaeger", "./jaeger"),
        Resource("otel-collector", "./otel-collector", {"OPENSEARCH_INITIAL_ADMIN_PASSWORD": "QWERqwer1!"}),
        # Resource("prometheus", "./prometheus"),
        Resource("opensearch", "./opensearch"),
        Resource("data-prepper", "./data-prepper"),
    ]

    if args.command == "up":
        create_docker_network()
        try:
            for resource in resources:
                up_docker_compose(resource)
        except subprocess.CalledProcessError as e:
            for resource in resources:
                down_docker_compose(resource)
                remove_docker_network()
            raise RuntimeError(f"docker compose up error {e}")

    elif args.command == "down":
        for resource in resources:
            down_docker_compose(resource)
        remove_docker_network()

    elif args.command == "jar":
        run_jar_with_javaagent()


if __name__ == "__main__":
    main()
