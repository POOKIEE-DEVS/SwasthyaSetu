"""WebSocket signalling relay and WebRTC ICE configuration."""

from __future__ import annotations

import base64
import hashlib
import hmac

from fastapi.testclient import TestClient

from app.core.config import settings
from app.realtime.webrtc import build_ice_configuration, consultation_room


def test_peer_joining_is_announced_to_the_room(client: TestClient) -> None:
    with client.websocket_connect("/ws/consultations/abc") as first:
        assert first.receive_json() == {
            "type": "joined",
            "room": "consultation:abc",
            "peers": 1,
        }

        with client.websocket_connect("/ws/consultations/abc") as second:
            assert second.receive_json()["peers"] == 2
            # The first peer learns the second arrived, which is its cue to
            # send an offer.
            assert first.receive_json() == {"type": "peer-joined", "peers": 2}


def test_offer_is_relayed_to_the_other_peer_only(client: TestClient) -> None:
    with (
        client.websocket_connect("/ws/consultations/xyz") as caller,
        client.websocket_connect("/ws/consultations/xyz") as callee,
    ):
        caller.receive_json()  # joined
        callee.receive_json()  # joined
        caller.receive_json()  # peer-joined

        offer = {"type": "offer", "payload": {"sdp": "v=0 fake-sdp"}}
        caller.send_json(offer)

        assert callee.receive_json() == offer


def test_unknown_signal_type_is_rejected_not_forwarded(client: TestClient) -> None:
    with (
        client.websocket_connect("/ws/consultations/guard") as sender,
        client.websocket_connect("/ws/consultations/guard") as other,
    ):
        sender.receive_json()
        other.receive_json()
        sender.receive_json()

        sender.send_json({"type": "definitely-not-a-signal"})

        error = sender.receive_json()
        assert error["type"] == "error"
        assert error["detail"] == "unsupported signal"

        # The far peer must not have been asked to interpret it. A valid
        # frame sent afterwards is what arrives instead.
        sender.send_json({"type": "hangup"})
        assert other.receive_json()["type"] == "hangup"


def test_rooms_are_isolated_from_each_other(client: TestClient) -> None:
    with (
        client.websocket_connect("/ws/consultations/room-a") as a,
        client.websocket_connect("/ws/consultations/room-b") as b,
    ):
        assert a.receive_json()["peers"] == 1
        # A second, separate consultation must not see the first one's peer.
        assert b.receive_json()["peers"] == 1


def test_ice_configuration_includes_stun_and_turn() -> None:
    config = build_ice_configuration()

    urls = [url for server in config.ice_servers for url in server.urls]
    assert any(url.startswith("stun:") for url in urls)
    assert any(url.startswith("turn:") for url in urls)


def test_turn_credentials_are_time_limited_hmacs() -> None:
    """The static secret must never be what a browser receives."""
    config = build_ice_configuration()
    turn = next(s for s in config.ice_servers if s.urls[0].startswith("turn:"))

    assert turn.username is not None and turn.credential is not None
    assert turn.credential != settings.turn_credential

    expiry, _, user = turn.username.partition(":")
    assert expiry.isdigit()
    assert user == settings.turn_username

    expected = base64.b64encode(
        hmac.new(
            settings.turn_credential.encode(),
            turn.username.encode(),
            hashlib.sha1,
        ).digest()
    ).decode()
    assert turn.credential == expected


def test_consultation_room_is_namespaced() -> None:
    assert consultation_room("123") == "consultation:123"
