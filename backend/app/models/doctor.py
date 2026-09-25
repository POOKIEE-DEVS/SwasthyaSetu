"""Doctor professional profile and post-consultation ratings."""

from __future__ import annotations

import uuid
from datetime import datetime

import sqlalchemy as sa
from sqlalchemy.dialects.postgresql import ARRAY
from sqlmodel import Field, SQLModel

from app.models.enums import LICENSE_STATUS, LicenseStatus
from app.models.mixins import (
    created_at_column,
    fk_column,
    new_uuid,
    pk_column,
    updated_at_column,
)


class DoctorProfile(SQLModel, table=True):
    """One-to-one with an account whose role is ``doctor``.

    ``license_status`` is the admin-approval workflow: doctors stay
    ``pending`` until an admin verifies their license.
    """

    __tablename__ = "doctor_profiles"
    __table_args__ = (
        sa.CheckConstraint(
            "experience_years >= 0", name="ck_doctor_profiles_experience_non_negative"
        ),
        # The matching pipeline only ever scans approved, available doctors,
        # so the index carries that predicate rather than the whole table.
        sa.Index(
            "idx_doctor_profiles_matchable",
            "specialization",
            postgresql_where=sa.text("is_available AND license_status = 'approved'"),
        ),
    )

    user_id: uuid.UUID = Field(
        sa_column=fk_column("users.id", primary_key=True, ondelete="CASCADE")
    )
    specialization: str = Field(sa_column=sa.Column(sa.Text, nullable=False))
    license_number: str = Field(
        sa_column=sa.Column(sa.Text, nullable=False, unique=True)
    )
    license_status: LicenseStatus = Field(
        default=LicenseStatus.PENDING,
        sa_column=sa.Column(
            LICENSE_STATUS, nullable=False, server_default=LicenseStatus.PENDING.value
        ),
    )
    # Languages the doctor can consult in; matched to users.preferred_language.
    languages: list[str] = Field(
        default_factory=list,
        sa_column=sa.Column(
            ARRAY(sa.Text), nullable=False, server_default=sa.text("'{}'")
        ),
    )
    experience_years: int = Field(
        default=0,
        sa_column=sa.Column(sa.Integer, nullable=False, server_default=sa.text("0")),
    )
    # Real-time availability toggle (Week 7); only available doctors match.
    is_available: bool = Field(
        default=False,
        sa_column=sa.Column(sa.Boolean, nullable=False, server_default=sa.false()),
    )
    created_at: datetime | None = Field(default=None, sa_column=created_at_column())
    updated_at: datetime | None = Field(default=None, sa_column=updated_at_column())


class DoctorRating(SQLModel, table=True):
    """Patient feedback after a consultation; one rating per consultation.

    ``doctor_id`` is duplicated from the consultation so per-doctor
    aggregation doesn't need a join on the hot path.
    """

    __tablename__ = "doctor_ratings"
    __table_args__ = (
        sa.CheckConstraint(
            "rating BETWEEN 1 AND 5", name="ck_doctor_ratings_rating_range"
        ),
        sa.Index("idx_doctor_ratings_doctor", "doctor_id"),
    )

    id: uuid.UUID = Field(default_factory=new_uuid, sa_column=pk_column())
    consultation_id: uuid.UUID = Field(
        sa_column=fk_column("consultations.id", ondelete="CASCADE", unique=True)
    )
    doctor_id: uuid.UUID = Field(sa_column=fk_column("users.id", ondelete="CASCADE"))
    rating: int = Field(sa_column=sa.Column(sa.SmallInteger, nullable=False))
    feedback: str | None = Field(default=None, sa_column=sa.Column(sa.Text))
    created_at: datetime | None = Field(default=None, sa_column=created_at_column())
