from __future__ import annotations

from fastapi.testclient import TestClient


def test_health(client: TestClient) -> None:
    body = client.get("/health").json()
    assert body["status"] == "ok"
    assert body["model_configured"] is False
    assert body["turn_configured"] is False


def test_health_under_api_prefix(client: TestClient) -> None:
    assert client.get("/api/v1/health").status_code == 200


def test_request_id_echoed(client: TestClient) -> None:
    response = client.get("/api/v1/health", headers={"X-Request-ID": "abc"})
    assert response.headers["X-Request-ID"] == "abc"
