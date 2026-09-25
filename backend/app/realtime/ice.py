"""ICE servers handed to each browser before it opens its peer connection.

STUN alone connects two laptops only when they can reach each other more or
less directly. Across different networks (venue Wi-Fi, mobile hotspots,
carrier NAT) the call needs TURN, a relay the media flows through when no
direct path exists. The media stays DTLS-SRTP encrypted end to end, but it
*does* transit the relay.

This backend cannot host TURN itself, because web hosts like Render expose
no UDP ports. It uses a hosted TURN service instead:

- Option A, Cloudflare: short-lived credentials are minted per join via their
  API, so no long-lived secret ever reaches a browser.
- Option B, static credentials from any TURN provider (e.g. Metered).
"""

from __future__ import annotations

import logging

import httpx

from app.core.config import settings
from app.schemas.consultation import IceServer

logger = logging.getLogger(__name__)

CLOUDFLARE_URL = (
    "https://rtc.live.cloudflare.com/v1/turn/keys/{key_id}/credentials/"
    "generate-ice-servers"
)


def _drop_port_53(urls: list[str]) -> list[str]:
    # Browsers block port 53, and those URLs just time out.
    return [u for u in urls if ":53?" not in u and not u.endswith(":53")]


async def _cloudflare_ice_servers(client: httpx.AsyncClient) -> list[IceServer]:
    response = await client.post(
        CLOUDFLARE_URL.format(key_id=settings.cloudflare_turn_key_id),
        headers={"Authorization": f"Bearer {settings.cloudflare_turn_api_token}"},
        json={"ttl": settings.turn_credential_ttl_seconds},
        timeout=10.0,
    )
    response.raise_for_status()
    servers = []
    for entry in response.json()["iceServers"]:
        urls = entry["urls"] if isinstance(entry["urls"], list) else [entry["urls"]]
        urls = _drop_port_53(urls)
        if urls:
            servers.append(
                IceServer(
                    urls=urls,
                    username=entry.get("username"),
                    credential=entry.get("credential"),
                )
            )
    return servers


def turn_configured() -> bool:
    return bool(
        (settings.cloudflare_turn_key_id and settings.cloudflare_turn_api_token)
        or settings.turn_url_list
    )


async def get_ice_servers(client: httpx.AsyncClient | None = None) -> list[IceServer]:
    """STUN always, plus TURN when configured.

    If the TURN provider fails, fall back to STUN only rather than failing the
    call. Same-network calls still connect, and the log says why others may
    not.
    """
    servers: list[IceServer] = []

    if settings.cloudflare_turn_key_id and settings.cloudflare_turn_api_token:
        try:
            if client is None:
                async with httpx.AsyncClient() as own:
                    return await _cloudflare_ice_servers(own)
            return await _cloudflare_ice_servers(client)
        except Exception as exc:
            logger.error("cloudflare TURN credential request failed", exc_info=exc)

    if settings.turn_url_list:
        servers.append(
            IceServer(
                urls=settings.turn_url_list,
                username=settings.turn_username or None,
                credential=settings.turn_credential or None,
            )
        )

    if settings.stun_url_list:
        servers.insert(0, IceServer(urls=settings.stun_url_list))
    return servers
