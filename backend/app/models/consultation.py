"""Live emergency consultations and scheduled appointments."""

from __future__ import annotations

import uuid
from datetime import datetime

import sqlalchemy as sa
from sqlmodel import Field, SQLModel

from app.models.enums import (
    APPOINTMENT_STATUS,
    CONSULTATION_STATUS,
    AppointmentStatus,
    ConsultationStatus,
)
from app.models.mixins import (
    created_at_column,
    fk_column,
    new_uuid,
    pk_column,
    updated_at_column,
)


class Consultation(SQLModel, table=True):
    """Emergency consultation record: who talked to whom, over which call
    room, and how it ended. Scheduled visits live in :class:`Appointment`.
    """

    __tablename__ = "consultations"
    __table_args__ = (
        sa.Index("idx_consultations_patient", "patient_id", sa.text("created_at DESC")),
        sa.Index("idx_consultations_doctor", "doctor_id", sa.text("created_at DESC")),
    )

    id: uuid.UUID = Field(default_factory=new_uuid, sa_column=pk_column())
    patient_id: uuid.UUID = Field(sa_column=fk_column("users.id", ondelete="CASCADE"))
    # Assigned by the matching service after creation, so nullable until then.
    doctor_id: uuid.UUID | None = Field(
        default=None,
        sa_column=fk_column("users.id", nullable=True, ondelete="SET NULL"),
    )
    status: ConsultationStatus = Field(
        default=ConsultationStatus.REQUESTED,
        sa_column=sa.Column(
            CONSULTATION_STATUS,
            nullable=False,
            server_default=ConsultationStatus.REQUESTED.value,
        ),
    )
    # Room identifier for the WebRTC session (Week 13); ICE servers are
    # issued separately per join, they are not stored here.
    call_link: str | None = Field(default=None, sa_column=sa.Column(sa.Text))
    started_at: datetime | None = Field(
        default=None, sa_column=sa.Column(sa.DateTime(timezone=True))
    )
    ended_at: datetime | None = Field(
        default=None, sa_column=sa.Column(sa.DateTime(timezone=True))
    )
    created_at: datetime | None = Field(default=None, sa_column=created_at_column())
    updated_at: datetime | None = Field(default=None, sa_column=updated_at_column())


class Appointment(SQLModel, table=True):
    """Scheduled (non-emergency) consultation."""

    __tablename__ = "appointments"
    __table_args__ = (
        sa.Index(
            "idx_appointments_patient", "patient_id", sa.text("scheduled_at DESC")
        ),
        sa.Index("idx_appointments_doctor", "doctor_id", sa.text("scheduled_at DESC")),
    )

    id: uuid.UUID = Field(default_factory=new_uuid, sa_column=pk_column())
    patient_id: uuid.UUID = Field(sa_column=fk_column("users.id", ondelete="CASCADE"))
    doctor_id: uuid.UUID | None = Field(
        default=None,
        sa_column=fk_column("users.id", nullable=True, ondelete="SET NULL"),
    )
    scheduled_at: datetime = Field(
        sa_column=sa.Column(sa.DateTime(timezone=True), nullable=False)
    )
    status: AppointmentStatus = Field(
        default=AppointmentStatus.SCHEDULED,
        sa_column=sa.Column(
            APPOINTMENT_STATUS,
            nullable=False,
            server_default=AppointmentStatus.SCHEDULED.value,
        ),
    )
    notes: str | None = Field(default=None, sa_column=sa.Column(sa.Text))
    created_at: datetime | None = Field(default=None, sa_column=created_at_column())
    updated_at: datetime | None = Field(default=None, sa_column=updated_at_column())
