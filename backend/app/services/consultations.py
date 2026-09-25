"""In-memory consultation registry.

State lives in this process only and is lost on restart, which is acceptable
for the demo. That is also why the backend must run as exactly one process:
a second worker would hold a separate registry and separate WebSocket rooms.

All methods are synchronous and never await, so on the single asyncio event
loop each call is atomic. Two doctors pressing "Accept" at once cannot both
win.
"""

from __future__ import annotations

import secrets
import time
from dataclasses import dataclass, field
from typing import Literal

from app.schemas.consultation import ConsultationPublic

Role = Literal["patient", "doctor"]
Status = Literal["waiting", "active", "ended"]


class ConsultationNotFoundError(Exception):
    pass


class ConsultationUnavailableError(Exception):
    """Already accepted by another doctor, or already ended."""


@dataclass
class Consultation:
    id: str
    patient_name: str
    summary: str | None
    patient_token: str
    doctor_token: str | None = None
    status: Status = "waiting"
    created_at: float = field(default_factory=time.time)
    updated_at: float = field(default_factory=time.time)

    def public(self) -> ConsultationPublic:
        return ConsultationPublic(
            id=self.id,
            patient_name=self.patient_name,
            summary=self.summary,
            status=self.status,
            created_at=self.created_at,
        )


class ConsultationRegistry:
    def __init__(self, ttl_seconds: float) -> None:
        self._items: dict[str, Consultation] = {}
        self._ttl = ttl_seconds

    def _purge_expired(self) -> None:
        cutoff = time.time() - self._ttl
        for cid in [c.id for c in self._items.values() if c.updated_at < cutoff]:
            del self._items[cid]

    def create(self, patient_name: str, summary: str | None) -> Consultation:
        self._purge_expired()
        consultation = Consultation(
            id=secrets.token_urlsafe(8),
            patient_name=patient_name.strip(),
            summary=summary.strip() if summary else None,
            patient_token=secrets.token_urlsafe(24),
        )
        self._items[consultation.id] = consultation
        return consultation

    def get(self, consultation_id: str) -> Consultation:
        consultation = self._items.get(consultation_id)
        if consultation is None:
            raise ConsultationNotFoundError(consultation_id)
        return consultation

    def waiting(self) -> list[Consultation]:
        self._purge_expired()
        return sorted(
            (c for c in self._items.values() if c.status == "waiting"),
            key=lambda c: c.created_at,
        )

    def accept(self, consultation_id: str) -> Consultation:
        consultation = self.get(consultation_id)
        if consultation.status != "waiting":
            raise ConsultationUnavailableError(consultation_id)
        consultation.status = "active"
        consultation.doctor_token = secrets.token_urlsafe(24)
        consultation.updated_at = time.time()
        return consultation

    def end(self, consultation_id: str, token: str) -> Consultation:
        consultation = self.get(consultation_id)
        if self.role_for(consultation, token) is None:
            raise ConsultationNotFoundError(consultation_id)
        consultation.status = "ended"
        consultation.updated_at = time.time()
        return consultation

    @staticmethod
    def role_for(consultation: Consultation, token: str) -> Role | None:
        """Which participant a token belongs to, compared in constant time."""
        if secrets.compare_digest(token, consultation.patient_token):
            return "patient"
        if consultation.doctor_token and secrets.compare_digest(
            token, consultation.doctor_token
        ):
            return "doctor"
        return None

    def clear(self) -> None:
        """Reset between rehearsals or tests."""
        self._items.clear()
