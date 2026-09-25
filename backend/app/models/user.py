"""Base account for every person on the platform: patient, doctor, or admin.

Doctor-specific fields live in :class:`~app.models.doctor.DoctorProfile`,
not here.
"""

from __future__ import annotations

import uuid
from datetime import datetime

import sqlalchemy as sa
from sqlmodel import Field, SQLModel

from app.models.enums import USER_ROLE, UserRole
from app.models.mixins import created_at_column, new_uuid, pk_column, updated_at_column


class User(SQLModel, table=True):
    __tablename__ = "users"

    id: uuid.UUID = Field(default_factory=new_uuid, sa_column=pk_column())
    full_name: str = Field(sa_column=sa.Column(sa.Text, nullable=False))
    email: str = Field(sa_column=sa.Column(sa.Text, nullable=False, unique=True))
    phone: str | None = Field(default=None, sa_column=sa.Column(sa.Text))
    password_hash: str = Field(sa_column=sa.Column(sa.Text, nullable=False))
    role: UserRole = Field(
        default=UserRole.PATIENT,
        sa_column=sa.Column(
            USER_ROLE, nullable=False, server_default=UserRole.PATIENT.value
        ),
    )
    # Matching ranks doctors by the patient's preferred language.
    preferred_language: str = Field(
        default="ne",
        sa_column=sa.Column(sa.Text, nullable=False, server_default="ne"),
    )
    # Free-form locality plus optional coordinates; proximity matching uses
    # whichever is available.
    address: str | None = Field(default=None, sa_column=sa.Column(sa.Text))
    location_lat: float | None = Field(
        default=None, sa_column=sa.Column(sa.Float(precision=53))
    )
    location_lng: float | None = Field(
        default=None, sa_column=sa.Column(sa.Float(precision=53))
    )
    is_active: bool = Field(
        default=True,
        sa_column=sa.Column(sa.Boolean, nullable=False, server_default=sa.true()),
    )
    created_at: datetime | None = Field(default=None, sa_column=created_at_column())
    updated_at: datetime | None = Field(default=None, sa_column=updated_at_column())
