"""Longer-term patient context: medical history and emergency contacts."""

from __future__ import annotations

import uuid
from datetime import date, datetime

import sqlalchemy as sa
from sqlmodel import Field, SQLModel

from app.models.mixins import (
    created_at_column,
    fk_column,
    new_uuid,
    pk_column,
    updated_at_column,
)


class MedicalHistory(SQLModel, table=True):
    """Chronic conditions and past diagnoses the AI engine references when
    assembling triage prompts. Distinct from the per-interaction
    triage_logs / symptom_reports series.
    """

    __tablename__ = "medical_history"
    __table_args__ = (sa.Index("idx_medical_history_patient", "patient_id"),)

    id: uuid.UUID = Field(default_factory=new_uuid, sa_column=pk_column())
    patient_id: uuid.UUID = Field(sa_column=fk_column("users.id", ondelete="CASCADE"))
    condition: str = Field(sa_column=sa.Column(sa.Text, nullable=False))
    notes: str | None = Field(default=None, sa_column=sa.Column(sa.Text))
    diagnosed_on: date | None = Field(default=None, sa_column=sa.Column(sa.Date))
    # Doctor or admin who entered the record, when it was not self-reported.
    recorded_by: uuid.UUID | None = Field(
        default=None,
        sa_column=fk_column("users.id", nullable=True, ondelete="SET NULL"),
    )
    created_at: datetime | None = Field(default=None, sa_column=created_at_column())
    updated_at: datetime | None = Field(default=None, sa_column=updated_at_column())


class EmergencyContact(SQLModel, table=True):
    """Patient-defined contacts alerted during a Red-level emergency, and
    shown offline alongside cached first-aid content (Week 18).
    """

    __tablename__ = "emergency_contacts"
    __table_args__ = (
        sa.Index("idx_emergency_contacts_patient", "patient_id", "sort_order"),
    )

    id: uuid.UUID = Field(default_factory=new_uuid, sa_column=pk_column())
    patient_id: uuid.UUID = Field(sa_column=fk_column("users.id", ondelete="CASCADE"))
    name: str = Field(sa_column=sa.Column(sa.Text, nullable=False))
    phone: str = Field(sa_column=sa.Column(sa.Text, nullable=False))
    relationship: str | None = Field(default=None, sa_column=sa.Column(sa.Text))
    # Lower numbers are called first.
    sort_order: int = Field(
        default=1,
        sa_column=sa.Column(
            sa.SmallInteger, nullable=False, server_default=sa.text("1")
        ),
    )
    created_at: datetime | None = Field(default=None, sa_column=created_at_column())
