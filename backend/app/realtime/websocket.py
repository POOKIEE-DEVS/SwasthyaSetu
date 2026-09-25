"""WebSocket endpoints: the doctor queue and per-consultation call signalling.

``/ws/doctors``
    Live list of waiting patients. A fresh snapshot is pushed on connect and
    after every change.

``/ws/consultations/{id}?token=...``
    WebRTC signalling for one call. The token decides the role (patient or
    doctor), and only those two tokens exist, so nobody else can join. Offer,
    answer, ICE candidates and hangup are relayed to the other participant.

Who makes the WebRTC offer: whoever is **already in the room** when the other
participant arrives. That participant receives ``peer-joined`` and makes the
offer. Only one side ever gets that event for a given arrival, so both
participants can never make an offer at the same time. The same rule
recovers from a page refresh: the returning participant arrives, and the one
who stayed makes a fresh offer.
"""

from __future__ import annotations

import contextlib
import logging

import anyio
from fastapi import APIRouter, WebSocket, WebSocketDisconnect
from pydantic import ValidationError

from app.realtime.connection_manager import manager
from app.schemas.consultation import SignalMessage
from app.services import consultations
from app.services.consultations import ConsultationNotFoundError

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/ws", tags=["realtime"])

DOCTOR_QUEUE_ROOM = "doctors:queue"

# Close codes the frontend distinguishes. 4000-4999 are application-defined.
CLOSE_UNAUTHORIZED = 4403
CLOSE_REPLACED = 4000

# The socket currently holding each (consultation, role) slot. A reconnect
# for the same role (a page refresh, or React's dev-mode double mount)
# replaces the old socket instead of being rejected.
_role_sockets: dict[tuple[str, str], WebSocket] = {}


def queue_snapshot() -> dict:
    return {
        "type": "queue",
        "consultations": [c.public().model_dump() for c in consultations.waiting()],
    }


async def broadcast_queue() -> None:
    await manager.broadcast(DOCTOR_QUEUE_ROOM, queue_snapshot())


@router.websocket("/doctors")
async def doctor_queue(websocket: WebSocket) -> None:
    await manager.connect(DOCTOR_QUEUE_ROOM, websocket)
    try:
        await manager.send(websocket, queue_snapshot())
        while True:
            # Nothing inbound is meaningful; receiving detects the disconnect.
            await websocket.receive_text()
    except WebSocketDisconnect:
        pass
    finally:
        with anyio.CancelScope(shield=True):
            await manager.disconnect(DOCTOR_QUEUE_ROOM, websocket)


@router.websocket("/consultations/{consultation_id}")
async def call_signalling(
    websocket: WebSocket, consultation_id: str, token: str = ""
) -> None:
    try:
        consultation = consultations.get(consultation_id)
    except ConsultationNotFoundError:
        await websocket.close(code=CLOSE_UNAUTHORIZED)
        return
    role = consultations.role_for(consultation, token) if token else None
    if role is None or consultation.status == "ended":
        await websocket.close(code=CLOSE_UNAUTHORIZED)
        return

    room = f"consultation:{consultation_id}"
    slot = (consultation_id, role)

    previous = _role_sockets.get(slot)
    _role_sockets[slot] = websocket
    if previous is not None:
        with contextlib.suppress(Exception):  # already gone
            await previous.close(code=CLOSE_REPLACED)

    await manager.connect(room, websocket)
    await manager.send(
        websocket,
        {
            "type": "joined",
            "role": role,
            "peer_present": _role_sockets.get(
                (consultation_id, "doctor" if role == "patient" else "patient")
            )
            is not None,
        },
    )
    await manager.broadcast(
        room, {"type": "peer-joined", "role": role}, exclude=websocket
    )

    try:
        while True:
            raw = await websocket.receive_json()
            try:
                signal = SignalMessage.model_validate(raw)
            except ValidationError:
                await manager.send(
                    websocket, {"type": "error", "detail": "unsupported signal"}
                )
                continue
            await manager.broadcast(
                room,
                {**signal.model_dump(mode="json"), "from": role},
                exclude=websocket,
            )
    except WebSocketDisconnect:
        pass
    except Exception as exc:
        logger.warning("signalling socket error", exc_info=exc)
    finally:
        # Shielded so cleanup still finishes if the handler is cancelled
        # rather than disconnected (e.g. on shutdown). Otherwise the other
        # participant would never hear "peer-left".
        with anyio.CancelScope(shield=True):
            await manager.disconnect(room, websocket)
            # A socket that was replaced must not announce "peer-left": the
            # participant is still here, on the new socket.
            if _role_sockets.get(slot) is websocket:
                del _role_sockets[slot]
                await manager.broadcast(room, {"type": "peer-left", "role": role})


def reset_realtime_state() -> None:
    """For tests."""
    _role_sockets.clear()
