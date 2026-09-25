"""Enumerations backed by native PostgreSQL enum types.

The ``name`` given to each SQLAlchemy ``Enum`` is the type name created in the
database, and ``values_callable`` makes Postgres store the lowercase *values*
below rather than the uppercase Python member names.
"""

from __future__ import annotations

import enum

import sqlalchemy as sa


class UserRole(enum.StrEnum):
    """Every account is exactly one of these (architecture §8)."""

    PATIENT = "patient"
    DOCTOR = "doctor"
    ADMIN = "admin"


class LicenseStatus(enum.StrEnum):
    """Admin approval workflow — doctors stay PENDING until verified."""

    PENDING = "pending"
    APPROVED = "approved"
    REJECTED = "rejected"
    SUSPENDED = "suspended"


class ConsultationStatus(enum.StrEnum):
    REQUESTED = "requested"
    MATCHED = "matched"
    IN_CALL = "in_call"
    COMPLETED = "completed"
    CANCELLED = "cancelled"


class AppointmentStatus(enum.StrEnum):
    SCHEDULED = "scheduled"
    COMPLETED = "completed"
    CANCELLED = "cancelled"
    NO_SHOW = "no_show"


class EmergencyLevel(enum.StrEnum):
    """Drives what the app shows after triage (architecture §4)."""

    GREEN = "green"
    YELLOW = "yellow"
    RED = "red"


class NotificationChannel(enum.StrEnum):
    EMAIL = "email"
    SMS = "sms"
    PUSH = "push"


class NotificationStatus(enum.StrEnum):
    PENDING = "pending"
    SENT = "sent"
    FAILED = "failed"


def pg_enum(python_enum: type[enum.Enum], name: str) -> sa.Enum:
    """Build the Postgres enum column type for ``python_enum``."""
    return sa.Enum(
        python_enum,
        name=name,
        values_callable=lambda members: [member.value for member in members],
    )


USER_ROLE = pg_enum(UserRole, "user_role")
LICENSE_STATUS = pg_enum(LicenseStatus, "license_status")
CONSULTATION_STATUS = pg_enum(ConsultationStatus, "consultation_status")
APPOINTMENT_STATUS = pg_enum(AppointmentStatus, "appointment_status")
EMERGENCY_LEVEL = pg_enum(EmergencyLevel, "emergency_level")
NOTIFICATION_CHANNEL = pg_enum(NotificationChannel, "notification_channel")
NOTIFICATION_STATUS = pg_enum(NotificationStatus, "notification_status")
