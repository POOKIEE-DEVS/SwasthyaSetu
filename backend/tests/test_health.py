"""Liveness and readiness behaviour."""

from __future__ import annotations

import pytest
from fastapi.testclient import TestClient

from app import __version__
from tests.conftest import build_app


def test_health_returns_ok(client: TestClient) -> None:
    response = client.get("/health")

    assert response.status_code == 200
    body = response.json()
    assert body["status"] == "ok"
    assert body["service"] == "backend"
    assert body["version"] == __version__
    assert body["uptime_seconds"] >= 0


def test_health_is_also_mounted_under_the_api_prefix(client: TestClient) -> None:
    assert client.get("/api/v1/health").status_code == 200


def test_readiness_reports_every_dependency(client: TestClient) -> None:
    response = client.get("/health/ready")

    assert response.status_code == 200
    body = response.json()
    assert body["status"] == "ready"
    assert set(body["dependencies"]) == {"postgres", "redis"}
    assert all(dep["status"] == "ok" for dep in body["dependencies"].values())


@pytest.mark.parametrize(
    ("kwargs", "broken"),
    [
        ({"postgres_fails": True}, "postgres"),
        ({"redis_fails": True}, "redis"),
    ],
)
def test_readiness_is_503_when_a_dependency_is_down(
    kwargs: dict[str, bool], broken: str
) -> None:
    with TestClient(build_app(**kwargs)) as client:
        response = client.get("/health/ready")

    assert response.status_code == 503
    body = response.json()
    assert body["status"] == "degraded"
    assert body["dependencies"][broken]["status"] == "error"
    assert body["dependencies"][broken]["detail"]


def test_liveness_stays_up_when_dependencies_are_down() -> None:
    """A Postgres outage must not get the container restarted."""
    with TestClient(build_app(postgres_fails=True, redis_fails=True)) as client:
        assert client.get("/health").status_code == 200


def test_request_id_is_echoed(client: TestClient) -> None:
    response = client.get("/health", headers={"X-Request-ID": "trace-me"})

    assert response.headers["X-Request-ID"] == "trace-me"


def test_request_id_is_generated_when_absent(client: TestClient) -> None:
    assert client.get("/health").headers["X-Request-ID"]


def test_unknown_route_returns_404(client: TestClient) -> None:
    assert client.get("/does-not-exist").status_code == 404
