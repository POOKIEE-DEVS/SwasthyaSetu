from __future__ import annotations

from fastapi.testclient import TestClient


def test_health(client: TestClient) -> None:
    body = client.get("/health").json()
    assert body["status"] == "ok"
    assert body["model_configured"] is False
    assert body["turn_configured"] is False
    assert body["database"] == "sqlite"
    assert body["google_sign_in_configured"] is False


def test_health_under_api_prefix(client: TestClient) -> None:
    assert client.get("/api/v1/health").status_code == 200


def test_request_id_echoed(client: TestClient) -> None:
    response = client.get("/api/v1/health", headers={"X-Request-ID": "abc"})
    assert response.headers["X-Request-ID"] == "abc"


def test_unreachable_database_does_not_stop_the_app(monkeypatch, fake_model) -> None:
    """A wrong DATABASE_URL must not take the emergency chat down with it."""
    from app import db
    from app.main import create_app

    broken = db.make_engine(
        "postgresql+psycopg://u:p@127.0.0.1:1/none?connect_timeout=1"
    )
    monkeypatch.setattr(db, "_engine", broken)
    monkeypatch.setattr(db, "_ready", False)

    with TestClient(create_app()) as client:
        health = client.get("/health").json()
        assert health["status"] == "ok"
        assert health["database_ready"] is False
        # Guests can still chat and request a doctor.
        chat = client.post(
            "/api/v1/chat", json={"messages": [{"role": "user", "content": "Burn"}]}
        )
        assert chat.status_code == 200
        ticket = client.post("/api/v1/consultations", json={"patient_name": "Ram"})
        assert ticket.status_code == 201
