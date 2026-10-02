"""Regression: plain comma-separated env values used to crash startup."""

from __future__ import annotations

import pytest

from app.core.config import Settings


def test_compose_style_list_values_parse(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("CORS_ORIGINS", "http://localhost:3000, https://demo.example")
    monkeypatch.setenv("TURN_URLS", "turn:turn.example:3478")
    monkeypatch.setenv("STUN_URLS", "stun:a:3478,stun:b:3478")

    s = Settings(_env_file=None)

    assert s.cors_origin_list == ["http://localhost:3000", "https://demo.example"]
    assert s.turn_url_list == ["turn:turn.example:3478"]
    assert s.stun_url_list == ["stun:a:3478", "stun:b:3478"]


def test_empty_list_values(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("CORS_ORIGINS", "")
    monkeypatch.setenv("TURN_URLS", "")

    s = Settings(_env_file=None)

    assert s.cors_origin_list == []
    assert s.turn_url_list == []


@pytest.mark.parametrize(
    ("raw", "expected", "kind"),
    [
        (
            "postgres://u:p@host/db?sslmode=require",
            "postgresql+psycopg://u:p@host/db?sslmode=require",
            "postgres",
        ),
        ("postgresql://u:p@host/db", "postgresql+psycopg://u:p@host/db", "postgres"),
        (" sqlite:///./local.db ", "sqlite:///./local.db", "sqlite"),
    ],
)
def test_database_url_gets_the_psycopg_driver(
    monkeypatch: pytest.MonkeyPatch, raw: str, expected: str, kind: str
) -> None:
    # Neon and Supabase hand out postgres:// URLs; a pasted value may carry
    # stray spaces (that exact mistake broke HF_SPACE_ID once).
    monkeypatch.setenv("DATABASE_URL", raw)
    s = Settings(_env_file=None)
    assert s.sqlalchemy_database_url == expected
    assert s.database_kind == kind


def test_dev_login_is_never_on_in_production(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("DEV_LOGIN", "true")
    monkeypatch.setenv("ENVIRONMENT", "production")
    assert Settings(_env_file=None).dev_login_enabled is False


def test_admin_emails_are_case_insensitive(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("ADMIN_EMAILS", "Admin@Example.com, second@example.com")
    s = Settings(_env_file=None)
    assert s.admin_email_list == ["admin@example.com", "second@example.com"]
