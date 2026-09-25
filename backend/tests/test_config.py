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
