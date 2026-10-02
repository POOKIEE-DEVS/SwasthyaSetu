"""Shared fixtures. No network: the model and TURN provider are faked."""

from __future__ import annotations

from collections.abc import Iterator

import pytest
from fastapi.testclient import TestClient
from sqlmodel import Session, SQLModel

from app import db
from app.ai.medgemma import medgemma
from app.core.config import settings
from app.main import create_app
from app.realtime.websocket import reset_realtime_state
from app.services import chat_rate_limiter, consultations
from app.services.auth import reset_pending_logins

ADMIN_EMAIL = "admin@example.com"


@pytest.fixture(autouse=True)
def clean_state(monkeypatch: pytest.MonkeyPatch) -> Iterator[None]:
    # No TURN provider, no static frontend, unless a test opts in.
    monkeypatch.setattr(settings, "cloudflare_turn_key_id", "")
    monkeypatch.setattr(settings, "cloudflare_turn_api_token", "")
    monkeypatch.setattr(settings, "turn_urls", "")
    monkeypatch.setattr(settings, "static_dir", "does-not-exist")
    # Never reach a real Space, even if a local backend/.env configures one.
    monkeypatch.setattr(medgemma, "_space_id", "")
    # A fresh in-memory database per test. Development login on, Google off.
    engine = db.make_engine("sqlite://")
    monkeypatch.setattr(db, "_engine", engine)
    from app import models  # noqa: F401

    SQLModel.metadata.create_all(engine)
    monkeypatch.setattr(settings, "environment", "development")
    monkeypatch.setattr(settings, "dev_login", True)
    monkeypatch.setattr(settings, "google_client_id", "")
    monkeypatch.setattr(settings, "google_client_secret", "")
    monkeypatch.setattr(settings, "public_url", "")
    monkeypatch.setattr(settings, "admin_emails", ADMIN_EMAIL)
    consultations.clear()
    chat_rate_limiter.clear()
    reset_realtime_state()
    reset_pending_logins()
    yield
    consultations.clear()
    chat_rate_limiter.clear()
    reset_realtime_state()


@pytest.fixture
def client() -> Iterator[TestClient]:
    with TestClient(create_app()) as test_client:
        yield test_client


@pytest.fixture
def fake_model(monkeypatch: pytest.MonkeyPatch) -> list[list[dict[str, str]]]:
    """Replace the Space call; records the messages each call was sent."""
    calls: list[list[dict[str, str]]] = []

    async def generate(messages: list[dict[str, str]]) -> str:
        calls.append(messages)
        return "1. Stay calm.\n2. Rest."

    monkeypatch.setattr(medgemma, "generate", generate)
    return calls


@pytest.fixture
def session() -> Iterator[Session]:
    with Session(db.get_engine()) as s:
        yield s


def sign_in(client: TestClient, email: str, name: str = "Test User") -> dict:
    """Development login; the client keeps the session cookie."""
    response = client.post(
        "/api/v1/auth/dev-login", json={"email": email, "name": name}
    )
    assert response.status_code == 200, response.text
    return response.json()
