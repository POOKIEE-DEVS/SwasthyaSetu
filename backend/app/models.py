"""Database tables.

Datetimes are stored as naive UTC (SQLite keeps no timezone) and sent to
clients as Unix timestamps, so browsers never misread them as local time.

No ``from __future__ import annotations`` here: SQLModel reads the field
annotations at class creation.
"""

from datetime import UTC, datetime

from sqlalchemy import Column, LargeBinary, Text
from sqlmodel import Field, SQLModel

PROFESSIONAL_ROLES = ("doctor", "pharmacist", "student")
ROLES = ("patient", *PROFESSIONAL_ROLES)


def utcnow() -> datetime:
    return datetime.now(UTC).replace(tzinfo=None)


def timestamp(value: datetime | None) -> float | None:
    return None if value is None else value.replace(tzinfo=UTC).timestamp()


class User(SQLModel, table=True):
    __tablename__ = "users"

    id: int | None = Field(default=None, primary_key=True)
    # Google's stable account id. Empty for development logins.
    google_sub: str | None = Field(default=None, unique=True, index=True)
    email: str = Field(unique=True, index=True, max_length=320)
    name: str = Field(max_length=120)
    picture_url: str | None = Field(default=None, max_length=1000)
    # patient | doctor | pharmacist | student. None until chosen after the
    # first sign-in. Admin access is not a role: see ADMIN_EMAILS.
    role: str | None = Field(default=None, max_length=20)
    created_at: datetime = Field(default_factory=utcnow)


class LoginSession(SQLModel, table=True):
    """A signed-in browser. Only a SHA-256 hash of the cookie is stored, so a
    database leak doesn't hand out working sessions."""

    __tablename__ = "sessions"

    token_hash: str = Field(primary_key=True, max_length=64)
    user_id: int = Field(foreign_key="users.id", index=True, ondelete="CASCADE")
    created_at: datetime = Field(default_factory=utcnow)
    expires_at: datetime


class Application(SQLModel, table=True):
    """A medical professional's verification request (KYC). One per user;
    a rejected applicant edits and resubmits the same application."""

    __tablename__ = "applications"

    id: int | None = Field(default=None, primary_key=True)
    user_id: int = Field(
        foreign_key="users.id", unique=True, index=True, ondelete="CASCADE"
    )
    role: str = Field(max_length=20)  # doctor | pharmacist | student
    full_name: str = Field(max_length=120)
    phone: str = Field(max_length=20)
    citizenship_number: str = Field(max_length=40)
    citizenship_district: str = Field(max_length=60)
    # Nepal Medical Council number (doctor) or Nepal Pharmacy Council
    # number (pharmacist).
    council_number: str | None = Field(default=None, max_length=40)
    # MBBS students: their college, and the doctor recommending them.
    institution: str | None = Field(default=None, max_length=160)
    recommender_name: str | None = Field(default=None, max_length=120)
    recommender_nmc: str | None = Field(default=None, max_length=40)
    status: str = Field(default="pending", max_length=20, index=True)
    rejection_reason: str | None = Field(default=None, max_length=500)
    submitted_at: datetime = Field(default_factory=utcnow)
    reviewed_at: datetime | None = None
    reviewed_by: str | None = Field(default=None, max_length=320)


class Document(SQLModel, table=True):
    """An uploaded verification document (photo or PDF).

    Stored in Postgres itself, not object storage: a handful of compressed
    photos per applicant fits the free database tier, and it means one less
    service to set up. Only the admin can read them back.
    """

    __tablename__ = "documents"

    id: int | None = Field(default=None, primary_key=True)
    application_id: int = Field(
        foreign_key="applications.id", index=True, ondelete="CASCADE"
    )
    kind: str = Field(max_length=40)
    # Detected from the file's bytes, never taken from the browser.
    content_type: str = Field(max_length=60)
    size: int
    data: bytes = Field(sa_column=Column(LargeBinary, nullable=False))
    created_at: datetime = Field(default_factory=utcnow)


class AuditEvent(SQLModel, table=True):
    """Who submitted, approved, or rejected an application, when, and why."""

    __tablename__ = "audit_events"

    id: int | None = Field(default=None, primary_key=True)
    application_id: int = Field(
        foreign_key="applications.id", index=True, ondelete="CASCADE"
    )
    actor_email: str = Field(max_length=320)
    action: str = Field(max_length=20)  # submitted | resubmitted | approved | rejected
    reason: str | None = Field(default=None, max_length=500)
    at: datetime = Field(default_factory=utcnow)


class Chat(SQLModel, table=True):
    """A signed-in patient's saved conversation with the AI assistant."""

    __tablename__ = "chats"

    # Random and unguessable, and every query also filters by owner.
    id: str = Field(primary_key=True, max_length=32)
    user_id: int = Field(foreign_key="users.id", index=True, ondelete="CASCADE")
    title: str = Field(max_length=80)
    created_at: datetime = Field(default_factory=utcnow)
    updated_at: datetime = Field(default_factory=utcnow, index=True)


class ChatEntry(SQLModel, table=True):
    __tablename__ = "chat_messages"

    id: int | None = Field(default=None, primary_key=True)
    chat_id: str = Field(foreign_key="chats.id", index=True, ondelete="CASCADE")
    role: str = Field(max_length=10)  # user | assistant
    content: str = Field(sa_column=Column(Text, nullable=False))
    created_at: datetime = Field(default_factory=utcnow)
