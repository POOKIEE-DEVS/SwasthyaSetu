from __future__ import annotations

import asyncio
from pathlib import Path

import httpx
import pytest
from fastapi.testclient import TestClient

from app.core.config import settings
from app.main import create_app
from app.realtime.ice import get_ice_servers

CLOUDFLARE_RESPONSE = {
    "iceServers": [
        {"urls": ["stun:stun.cloudflare.com:3478", "stun:stun.cloudflare.com:53"]},
        {
            "urls": [
                "turn:turn.cloudflare.com:3478?transport=udp",
                "turn:turn.cloudflare.com:53?transport=udp",
                "turns:turn.cloudflare.com:443?transport=tcp",
            ],
            "username": "u",
            "credential": "c",
        },
    ]
}


def test_stun_only_by_default() -> None:
    servers = asyncio.run(get_ice_servers())
    assert len(servers) == 1
    assert all(u.startswith("stun:") for u in servers[0].urls)


def test_static_turn_credentials(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(settings, "turn_urls", "turn:relay.example:3478")
    monkeypatch.setattr(settings, "turn_username", "demo")
    monkeypatch.setattr(settings, "turn_credential", "secret")

    servers = asyncio.run(get_ice_servers())

    assert servers[0].urls[0].startswith("stun:")
    assert servers[1].urls == ["turn:relay.example:3478"]
    assert (servers[1].username, servers[1].credential) == ("demo", "secret")


def test_cloudflare_credentials_drop_port_53(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(settings, "cloudflare_turn_key_id", "key")
    monkeypatch.setattr(settings, "cloudflare_turn_api_token", "token")
    seen = {}

    def handler(request: httpx.Request) -> httpx.Response:
        seen["auth"] = request.headers["Authorization"]
        seen["url"] = str(request.url)
        return httpx.Response(201, json=CLOUDFLARE_RESPONSE)

    async def run():
        async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as c:
            return await get_ice_servers(c)

    servers = asyncio.run(run())

    assert seen["auth"] == "Bearer token"
    assert "/keys/key/" in seen["url"]
    all_urls = [u for s in servers for u in s.urls]
    assert not any(":53" in u for u in all_urls)
    assert servers[1].username == "u"


def test_cloudflare_failure_falls_back_to_stun(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(settings, "cloudflare_turn_key_id", "key")
    monkeypatch.setattr(settings, "cloudflare_turn_api_token", "token")

    async def run():
        transport = httpx.MockTransport(lambda r: httpx.Response(500))
        async with httpx.AsyncClient(transport=transport) as c:
            return await get_ice_servers(c)

    servers = asyncio.run(run())
    assert [u.startswith("stun:") for s in servers for u in s.urls] == [True, True]


def test_frontend_served_from_same_origin(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    (tmp_path / "index.html").write_text("<h1>home</h1>")
    (tmp_path / "patient").mkdir()
    (tmp_path / "patient" / "index.html").write_text("<h1>patient</h1>")
    (tmp_path / "manifest.webmanifest").write_text("{}")
    monkeypatch.setattr(settings, "static_dir", str(tmp_path))

    with TestClient(create_app()) as client:
        assert "home" in client.get("/").text
        assert "patient" in client.get("/patient/").text
        assert (
            client.get("/manifest.webmanifest")
            .headers["content-type"]
            .startswith("application/manifest+json")
        )
        # API routes still win over the static mount.
        assert client.get("/api/v1/health").json()["status"] == "ok"
