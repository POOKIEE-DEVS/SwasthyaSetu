"""WebRTC teleconsultation support: ICE configuration and signalling.

Audio and video flow peer-to-peer between patient and doctor; the server's
only role is to (a) tell each peer which STUN/TURN servers to try and
(b) relay the SDP offer/answer and ICE candidates that let the peers find
each other. Media never transits the backend, which is what keeps a video
call cheap to serve and private by construction.

STUN is enough when both peers can be reached directly. TURN is the fallback
that relays media when they cannot -- symmetric NAT and restrictive mobile
carrier networks are common in the areas SwasthyaSetu targets, so a TURN
server is a requirement here rather than an optimisation.
"""

from __future__ import annotations

import base64
import hashlib
import hmac
import time
from typing import Any, Literal

from pydantic import BaseModel

from app.core.config import settings

# The message types relayed between peers. Anything else on a signalling
# socket is rejected rather than forwarded.
SignalType = Literal["offer", "answer", "ice-candidate", "hangup"]
SIGNAL_TYPES: frozenset[str] = frozenset(("offer", "answer", "ice-candidate", "hangup"))


class IceServer(BaseModel):
    """One entry of the ``iceServers`` array an RTCPeerConnection is built with."""

    urls: list[str]
    username: str | None = None
    credential: str | None = None


class IceConfiguration(BaseModel):
    """Everything a browser needs to open its peer connection."""

    ice_servers: list[IceServer]
    expires_at: int


class SignalMessage(BaseModel):
    """A signalling frame relayed verbatim to the other peer.

    ``payload`` is deliberately untyped: SDP blobs and ICE candidates are
    browser-defined structures that the server has no business interpreting.
    """

    type: SignalType
    payload: dict[str, Any] | None = None


def _turn_credentials(ttl_seconds: int) -> tuple[str, str]:
    """Mint short-lived TURN credentials (the ``coturn`` REST auth scheme).

    The username is ``<expiry-timestamp>:<user>`` and the password is its
    HMAC-SHA1 under a shared secret. This is why the static
    ``TURN_CREDENTIAL`` never has to be handed to a browser: each client gets
    a credential that stops working on its own.
    """
    expiry = int(time.time()) + ttl_seconds
    username = f"{expiry}:{settings.turn_username}"
    digest = hmac.new(
        settings.turn_credential.encode("utf-8"),
        username.encode("utf-8"),
        hashlib.sha1,
    ).digest()
    return username, base64.b64encode(digest).decode("ascii")


def build_ice_configuration() -> IceConfiguration:
    """Assemble the ICE server list for a client about to join a call."""
    ttl = settings.turn_credential_ttl_seconds
    username, credential = _turn_credentials(ttl)

    servers = [IceServer(urls=list(settings.stun_urls))]
    if settings.turn_urls:
        servers.append(
            IceServer(
                urls=list(settings.turn_urls),
                username=username,
                credential=credential,
            )
        )

    return IceConfiguration(ice_servers=servers, expires_at=int(time.time()) + ttl)


def consultation_room(consultation_id: str) -> str:
    """Room key for a consultation's signalling channel."""
    return f"consultation:{consultation_id}"
