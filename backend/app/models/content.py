"""Curated first-aid and medical reference content."""

from __future__ import annotations

import uuid
from datetime import datetime

import sqlalchemy as sa
from sqlmodel import Field, SQLModel

from app.models.mixins import (
    created_at_column,
    fk_column,
    new_uuid,
    pk_column,
    updated_at_column,
)


class Article(SQLModel, table=True):
    """First-aid reference content (WHO, Red Cross, Nepal Health Protocol,
    project handbook). Serves the public articles API, the offline cache, and
    -- from Week 11 -- retrieval grounding for the triage pipeline.
    """

    __tablename__ = "articles"
    __table_args__ = (
        sa.Index(
            "idx_articles_published",
            "category",
            "language",
            postgresql_where=sa.text("published"),
        ),
    )

    id: uuid.UUID = Field(default_factory=new_uuid, sa_column=pk_column())
    title: str = Field(sa_column=sa.Column(sa.Text, nullable=False))
    slug: str = Field(sa_column=sa.Column(sa.Text, nullable=False, unique=True))
    content: str = Field(sa_column=sa.Column(sa.Text, nullable=False))
    # Traceability: guidance must always point back to a real source.
    source_name: str = Field(sa_column=sa.Column(sa.Text, nullable=False))
    source_url: str | None = Field(default=None, sa_column=sa.Column(sa.Text))
    category: str | None = Field(default=None, sa_column=sa.Column(sa.Text))
    language: str = Field(
        default="en",
        sa_column=sa.Column(sa.Text, nullable=False, server_default="en"),
    )
    # Admins draft and review before content becomes publicly visible.
    published: bool = Field(
        default=False,
        sa_column=sa.Column(sa.Boolean, nullable=False, server_default=sa.false()),
    )
    created_by: uuid.UUID | None = Field(
        default=None,
        sa_column=fk_column("users.id", nullable=True, ondelete="SET NULL"),
    )
    created_at: datetime | None = Field(default=None, sa_column=created_at_column())
    updated_at: datetime | None = Field(default=None, sa_column=updated_at_column())
