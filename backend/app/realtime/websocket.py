"""Native FastAPI WebSocket endpoints.

Three channels, all backed by the shared
:class:`~app.realtime.connection_manager.ConnectionManager`:

``/ws/consultations/{id}``
    WebRTC signalling relay for one call -- offer, answer, ICE candidates,
    hangup -- forwarded to the other peer and nobody else.
``/ws/notifications/{user_id}``
    Server-to-client push for one account (doctor accepted, call reminder,
    emergency alert, consultation finished). Wired to the notification
    service in Week 14.
``/ws/doctors/queue``
    The shared queue every on-call doctor watches for incoming emergency
    requests. Wired to matching in Week 16.

Security note: these connections are unauthenticated in Week 1. JWT
verification on the handshake arrives in Week 5 alongside the HTTP
middleware -- until the auth service exists there is no token to check. The
seam is :func:`_authenticate`, which every endpoint already calls.
"""

from __future__ import annotations

import logging

from fastapi import APIRouter, WebSocket, WebSocketDisconnect
from pydantic import ValidationError

from app.realtime.connection_manager import manager
from app.realtime.webrtc import SignalMessage, consultation_room

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/ws", tags=["realtime"])


async def _authenticate(websocket: WebSocket) -> None:
    """Verify the connecting client (Week 5).

    Browsers cannot set headers on a WebSocket handshake, so the access
    token will arrive as a ``?token=`` query parameter and be validated with
    ``app.core.security.decode_token`` before the socket is accepted.
    """
    return None


@router.websocket("/consultations/{consultation_id}")
async def consultation_signalling(websocket: WebSocket, consultation_id: str) -> None:
    """Relay WebRTC signalling between the two peers of one consultation."""
    await _authenticate(websocket)
    room = consultation_room(consultation_id)
    await manager.connect(room, websocket)

    # Tell the joiner whether the other peer is already here, so the client
    # knows whether to send an offer or wait for one.
    await manager.send(
        websocket,
        {"type": "joined", "room": room, "peers": manager.room_size(room)},
    )
    await manager.broadcast(
        room,
        {"type": "peer-joined", "peers": manager.room_size(room)},
        exclude=websocket,
    )

    try:
        while True:
            raw = await websocket.receive_json()
            try:
                signal = SignalMessage.model_validate(raw)
            except ValidationError as exc:
                # Reject rather than forward: an unrecognised frame on a
                # signalling channel is a bug or an attack, never traffic
                # the far peer should be asked to interpret.
                await manager.send(
                    websocket,
                    {
                        "type": "error",
                        "detail": "unsupported signal",
                        "errors": exc.errors(),
                    },
                )
                continue

            await manager.broadcast(
                room, signal.model_dump(mode="json"), exclude=websocket
            )
    except WebSocketDisconnect:
        pass
    finally:
        await manager.disconnect(room, websocket)
        await manager.broadcast(
            room, {"type": "peer-left", "peers": manager.room_size(room)}
        )


@router.websocket("/notifications/{user_id}")
async def notification_stream(websocket: WebSocket, user_id: str) -> None:
    """Push channel for one account. Server-to-client only."""
    await _authenticate(websocket)
    room = f"user:{user_id}"
    await manager.connect(room, websocket)
    try:
        while True:
            # Nothing inbound is meaningful here; receiving is how a
            # disconnect is detected.
            await websocket.receive_text()
    except WebSocketDisconnect:
        pass
    finally:
        await manager.disconnect(room, websocket)


@router.websocket("/doctors/queue")
async def doctor_queue(websocket: WebSocket) -> None:
    """Live queue of emergency requests, watched by on-call doctors."""
    await _authenticate(websocket)
    room = "doctors:available"
    await manager.connect(room, websocket)
    try:
        while True:
            await websocket.receive_text()
    except WebSocketDisconnect:
        pass
    finally:
        await manager.disconnect(room, websocket)
