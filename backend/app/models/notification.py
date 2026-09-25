"""Delivery record for every outbound notification."""

from __future__ import annotations

import uuid
from datetime import datetime
from typing import Any

import sqlalchemy as sa
from sqlalchemy.dialects.postgresql import JSONB
from sqlmodel import Field, SQLModel

from app.models.enums import (
    NOTIFICATION_CHANNEL,
    NOTIFICATION_STATUS,
    NotificationChannel,
    NotificationStatus,
)
from app.models.mixins import created_at_column, fk_column, new_uuid, pk_column


class Notification(SQLModel, table=True):
    """One row per email, SMS, or push message, so the Celery dispatcher can
    retry failures and admins can audit what was sent.

    ``event_type`` stays free-form text because the set of events grows week
    by week (doctor_accepted, call_reminder, emergency_alert,
    consultation_finished, ...).
    """

    __tablename__ = "notifications"
    __table_args__ = (
        sa.Index("idx_notifications_user", "user_id", sa.text("created_at DESC")),
        # The dispatcher polls only undelivered rows.
        sa.Index(
            "idx_notifications_pending",
            "created_at",
            postgresql_where=sa.text("status = 'pending'"),
        ),
    )

    id: uuid.UUID = Field(default_factory=new_uuid, sa_column=pk_column())
    user_id: uuid.UUID = Field(sa_column=fk_column("users.id", ondelete="CASCADE"))
    channel: NotificationChannel = Field(
        sa_column=sa.Column(NOTIFICATION_CHANNEL, nullable=False)
    )
    event_type: str = Field(sa_column=sa.Column(sa.Text, nullable=False))
    # Channel-specific message content and metadata.
    payload: dict[str, Any] | None = Field(default=None, sa_column=sa.Column(JSONB))
    status: NotificationStatus = Field(
        default=NotificationStatus.PENDING,
        sa_column=sa.Column(
            NOTIFICATION_STATUS,
            nullable=False,
            server_default=NotificationStatus.PENDING.value,
        ),
    )
    sent_at: datetime | None = Field(
        default=None, sa_column=sa.Column(sa.DateTime(timezone=True))
    )
    created_at: datetime | None = Field(default=None, sa_column=created_at_column())
