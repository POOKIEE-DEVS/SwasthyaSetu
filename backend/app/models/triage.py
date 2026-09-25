"""AI triage interactions and the patient symptom series behind them.

Together these two tables are the conversational memory described in
architecture §3 — always scoped to a single patient, never shared across
accounts.
"""

from __future__ import annotations

import uuid
from datetime import datetime
from decimal import Decimal
from typing import Any

import sqlalchemy as sa
from sqlalchemy.dialects.postgresql import JSONB
from sqlmodel import Field, SQLModel

from app.models.enums import EMERGENCY_LEVEL, EmergencyLevel
from app.models.mixins import created_at_column, fk_column, new_uuid, pk_column


class TriageLog(SQLModel, table=True):
    """One row per AI triage interaction: what the patient described, what
    the pipeline concluded, and whether it escalated to a live consultation.
    """

    __tablename__ = "triage_logs"
    __table_args__ = (
        sa.CheckConstraint(
            "confidence IS NULL OR (confidence >= 0 AND confidence <= 100)",
            name="ck_triage_logs_confidence_range",
        ),
        # Conversational memory reads a patient's recent history, newest first.
        sa.Index("idx_triage_logs_patient", "patient_id", sa.text("created_at DESC")),
    )

    id: uuid.UUID = Field(default_factory=new_uuid, sa_column=pk_column())
    patient_id: uuid.UUID = Field(sa_column=fk_column("users.id", ondelete="CASCADE"))
    # The patient's free-text description, kept verbatim for audit.
    input_text: str = Field(sa_column=sa.Column(sa.Text, nullable=False))
    # Structured symptoms extracted by the pipeline (Week 10); the shape will
    # evolve with the extractor, hence JSONB rather than columns.
    symptoms: dict[str, Any] | None = Field(default=None, sa_column=sa.Column(JSONB))
    emergency_level: EmergencyLevel | None = Field(
        default=None, sa_column=sa.Column(EMERGENCY_LEVEL)
    )
    # Model confidence as a percentage, e.g. 94.00.
    confidence: Decimal | None = Field(
        default=None, sa_column=sa.Column(sa.Numeric(5, 2))
    )
    requires_doctor: bool = Field(
        default=False,
        sa_column=sa.Column(sa.Boolean, nullable=False, server_default=sa.false()),
    )
    suggested_action: str | None = Field(default=None, sa_column=sa.Column(sa.Text))
    # The safety-validated response actually shown to the patient.
    response: str | None = Field(default=None, sa_column=sa.Column(sa.Text))
    escalated_to_consultation_id: uuid.UUID | None = Field(
        default=None,
        sa_column=fk_column("consultations.id", nullable=True, ondelete="SET NULL"),
    )
    created_at: datetime | None = Field(default=None, sa_column=created_at_column())


class SymptomReport(SQLModel, table=True):
    """Individual symptom entries linked to a patient over time — the raw
    series feeding conversational memory ("fever yesterday, breathing
    difficulty today"). Reports logged offline are queued on the client and
    synced later (Week 18), hence ``reported_offline``.
    """

    __tablename__ = "symptom_reports"
    __table_args__ = (
        sa.Index(
            "idx_symptom_reports_patient", "patient_id", sa.text("created_at DESC")
        ),
    )

    id: uuid.UUID = Field(default_factory=new_uuid, sa_column=pk_column())
    patient_id: uuid.UUID = Field(sa_column=fk_column("users.id", ondelete="CASCADE"))
    # Set when the report came out of a triage session; offline-synced
    # reports may not have one.
    triage_log_id: uuid.UUID | None = Field(
        default=None,
        sa_column=fk_column("triage_logs.id", nullable=True, ondelete="SET NULL"),
    )
    symptom: str = Field(sa_column=sa.Column(sa.Text, nullable=False))
    description: str | None = Field(default=None, sa_column=sa.Column(sa.Text))
    reported_offline: bool = Field(
        default=False,
        sa_column=sa.Column(sa.Boolean, nullable=False, server_default=sa.false()),
    )
    # When the patient says it started, as opposed to when it was recorded.
    onset_at: datetime | None = Field(
        default=None, sa_column=sa.Column(sa.DateTime(timezone=True))
    )
    created_at: datetime | None = Field(default=None, sa_column=created_at_column())
