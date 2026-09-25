from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, Field


class IceServer(BaseModel):
    """One entry of RTCPeerConnection's ``iceServers`` option."""

    urls: list[str]
    username: str | None = None
    credential: str | None = None


class ConsultationCreate(BaseModel):
    patient_name: str = Field(min_length=1, max_length=60)
    # The patient's chat with the AI, shared with the doctor only when the
    # patient ticks the consent box.
    summary: str | None = Field(default=None, max_length=6000)


class ConsultationPublic(BaseModel):
    """What the doctor queue sees. Never includes tokens."""

    id: str
    patient_name: str
    summary: str | None
    status: Literal["waiting", "active", "ended"]
    created_at: float


class CallTicket(BaseModel):
    """Everything one participant needs to join the call."""

    consultation: ConsultationPublic
    role: Literal["patient", "doctor"]
    # Room-scoped secret for the signalling WebSocket. Only the two
    # participants ever hold one, which is what stops a third person joining.
    token: str
    ice_servers: list[IceServer]


class EndCall(BaseModel):
    token: str


class SignalMessage(BaseModel):
    """A signalling frame relayed verbatim to the other participant."""

    type: Literal["offer", "answer", "ice-candidate", "hangup"]
    payload: dict[str, Any] | None = None
