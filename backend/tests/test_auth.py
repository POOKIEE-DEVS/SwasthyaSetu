from __future__ import annotations

import base64
import json
import time
from urllib.parse import parse_qs, urlparse

import pytest
from fastapi.testclient import TestClient

from app.core.config import settings
from app.services import auth as auth_service
from tests.conftest import ADMIN_EMAIL, sign_in


def fake_id_token(**overrides: object) -> str:
    claims = {
        "iss": "https://accounts.google.com",
        "aud": "client-123",
        "exp": time.time() + 600,
        "sub": "google-sub-1",
        "email": "Sita@Example.com",
        "email_verified": True,
        "name": "Sita Sharma",
        "picture": "https://example.com/p.png",
        **overrides,
    }
    payload = base64.urlsafe_b64encode(json.dumps(claims).encode()).rstrip(b"=")
    return f"header.{payload.decode()}.signature"


@pytest.fixture
def google(monkeypatch: pytest.MonkeyPatch) -> list[dict]:
    """Google configured, with the token endpoint faked."""
    monkeypatch.setattr(settings, "google_client_id", "client-123")
    monkeypatch.setattr(settings, "google_client_secret", "secret")
    exchanges: list[dict] = []

    async def exchange(code: str, verifier: str, redirect_uri: str) -> dict:
        exchanges.append({"code": code, "verifier": verifier, "uri": redirect_uri})
        return {"id_token": fake_id_token()}

    monkeypatch.setattr("app.api.auth.exchange_code", exchange)
    return exchanges


def start_google(client: TestClient, next_path: str = "/") -> tuple[str, dict]:
    start = client.get(
        f"/api/v1/auth/google/login?next={next_path}", follow_redirects=False
    )
    assert start.status_code == 303
    consent = urlparse(start.headers["location"])
    assert consent.netloc == "accounts.google.com"
    params = parse_qs(consent.query)
    return params["state"][0], params


def test_signed_out_me(client: TestClient) -> None:
    body = client.get("/api/v1/auth/me").json()
    assert body == {"user": None, "google_enabled": False, "dev_login": True}


def test_dev_login_session_and_logout(client: TestClient) -> None:
    sign_in(client, "ram@example.com", "Ram")
    me = client.get("/api/v1/auth/me").json()["user"]
    assert me["email"] == "ram@example.com"
    assert me["role"] is None
    assert me["is_admin"] is False

    assert client.post("/api/v1/auth/logout").status_code == 204
    assert client.get("/api/v1/auth/me").json()["user"] is None


def test_dev_login_is_off_in_production(
    client: TestClient, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setattr(settings, "environment", "production")
    response = client.post(
        "/api/v1/auth/dev-login", json={"email": "x@example.com", "name": "X"}
    )
    assert response.status_code == 404


def test_admin_comes_from_admin_emails(client: TestClient) -> None:
    assert sign_in(client, ADMIN_EMAIL.upper())["is_admin"] is True


def test_choose_role(client: TestClient) -> None:
    assert client.post("/api/v1/auth/role", json={"role": "patient"}).status_code == 401
    sign_in(client, "gita@example.com")
    chosen = client.post("/api/v1/auth/role", json={"role": "patient"})
    assert chosen.json()["role"] == "patient"
    bad = client.post("/api/v1/auth/role", json={"role": "admin"})
    assert bad.status_code == 422


def test_google_login_needs_configuration(client: TestClient) -> None:
    response = client.get("/api/v1/auth/google/login", follow_redirects=False)
    assert response.status_code == 404


def test_google_round_trip(client: TestClient, google: list[dict]) -> None:
    state, params = start_google(client, "/doctor/")
    assert params["code_challenge_method"] == ["S256"]
    assert params["scope"] == ["openid email profile"]
    redirect_uri = params["redirect_uri"][0]
    assert redirect_uri == "http://testserver/api/v1/auth/google/callback"

    back = client.get(
        f"/api/v1/auth/google/callback?code=abc&state={state}", follow_redirects=False
    )

    # New user: pick a role first, then continue to /doctor/.
    assert back.status_code == 303
    assert back.headers["location"] == "/account/?next=%2Fdoctor%2F"
    assert google[0]["code"] == "abc"
    assert google[0]["uri"] == redirect_uri
    me = client.get("/api/v1/auth/me").json()
    assert me["google_enabled"] is True
    assert me["user"]["email"] == "sita@example.com"  # normalised
    assert me["user"]["name"] == "Sita Sharma"


def test_returning_user_goes_straight_to_next(
    client: TestClient, google: list[dict]
) -> None:
    for _ in range(2):
        state, _ = start_google(client)
        back = client.get(
            f"/api/v1/auth/google/callback?code=c&state={state}",
            follow_redirects=False,
        )
        client.post("/api/v1/auth/role", json={"role": "patient"})
    assert back.headers["location"] == "/"


def test_callback_rejects_state_from_another_browser(
    client: TestClient, google: list[dict]
) -> None:
    state, _ = start_google(client)
    client.cookies.clear()  # the callback arrives without our state cookie

    back = client.get(
        f"/api/v1/auth/google/callback?code=c&state={state}", follow_redirects=False
    )

    assert back.headers["location"].startswith("/account/?error=")
    assert google == []  # the code was never exchanged
    assert client.get("/api/v1/auth/me").json()["user"] is None


def test_cancelled_consent(client: TestClient, google: list[dict]) -> None:
    state, _ = start_google(client)
    back = client.get(
        f"/api/v1/auth/google/callback?error=access_denied&state={state}",
        follow_redirects=False,
    )
    assert "cancelled" in back.headers["location"]


@pytest.mark.parametrize(
    "overrides",
    [
        {"iss": "https://evil.example"},
        {"aud": "someone-else"},
        {"exp": time.time() - 10},
        {"email_verified": False},
    ],
)
def test_id_token_claims_are_checked(
    monkeypatch: pytest.MonkeyPatch, overrides: dict
) -> None:
    monkeypatch.setattr(settings, "google_client_id", "client-123")
    with pytest.raises(auth_service.OAuthError):
        auth_service.profile_from_id_token(fake_id_token(**overrides))


@pytest.mark.parametrize(
    ("value", "expected"),
    [
        ("/doctor/", "/doctor/"),
        ("https://evil.example", "/"),
        ("//evil.example", "/"),
        ("/\\evil.example", "/"),
        (None, "/"),
    ],
)
def test_next_is_same_site_only(value: str | None, expected: str) -> None:
    assert auth_service.safe_next(value) == expected
