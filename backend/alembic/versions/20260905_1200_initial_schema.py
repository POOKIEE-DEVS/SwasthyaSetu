"""Initial schema: the eleven tables of architecture section 6.

Written in Week 1 as the reviewable schema of record; first applied in
Week 2. Table order follows foreign-key dependencies so the migration
applies cleanly to an empty database, and ``downgrade`` reverses it exactly.

Enum types are created explicitly before the tables that use them and
dropped explicitly afterwards. Letting ``create_table`` create them
implicitly works, but leaves the types behind on downgrade, which makes a
re-upgrade fail with "type already exists".

Revision ID: 0001_initial_schema
Revises:
Create Date: 2026-09-05 12:00:00+00:00
"""

from __future__ import annotations

from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op
from sqlalchemy.dialects import postgresql

revision: str = "0001_initial_schema"
down_revision: str | None = None
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


# create_type=False: the types are created and dropped explicitly below, so
# create_table must not try to create them a second time.
user_role = postgresql.ENUM(
    "patient", "doctor", "admin", name="user_role", create_type=False
)
license_status = postgresql.ENUM(
    "pending", "approved", "rejected", "suspended",
    name="license_status", create_type=False,
)
consultation_status = postgresql.ENUM(
    "requested", "matched", "in_call", "completed", "cancelled",
    name="consultation_status", create_type=False,
)
appointment_status = postgresql.ENUM(
    "scheduled", "completed", "cancelled", "no_show",
    name="appointment_status", create_type=False,
)
emergency_level = postgresql.ENUM(
    "green", "yellow", "red", name="emergency_level", create_type=False
)
notification_channel = postgresql.ENUM(
    "email", "sms", "push", name="notification_channel", create_type=False
)
notification_status = postgresql.ENUM(
    "pending", "sent", "failed", name="notification_status", create_type=False
)

ENUM_TYPES = (
    user_role,
    license_status,
    consultation_status,
    appointment_status,
    emergency_level,
    notification_channel,
    notification_status,
)


def upgrade() -> None:
    bind = op.get_bind()
    for enum_type in ENUM_TYPES:
        enum_type.create(bind, checkfirst=True)

    # --- users ---------------------------------------------------------
    # Base account for every person: patient, doctor, or admin.
    op.create_table(
        "users",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            server_default=sa.text("gen_random_uuid()"),
            nullable=False,
        ),
        sa.Column("full_name", sa.Text(), nullable=False),
        sa.Column("email", sa.Text(), nullable=False),
        sa.Column("phone", sa.Text(), nullable=True),
        sa.Column("password_hash", sa.Text(), nullable=False),
        sa.Column("role", user_role, server_default="patient", nullable=False),
        # Matching ranks doctors by the patient's preferred language.
        sa.Column(
            "preferred_language", sa.Text(), server_default="ne", nullable=False
        ),
        sa.Column("address", sa.Text(), nullable=True),
        sa.Column("location_lat", sa.Float(precision=53), nullable=True),
        sa.Column("location_lng", sa.Float(precision=53), nullable=True),
        sa.Column("is_active", sa.Boolean(), server_default=sa.true(), nullable=False),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_users")),
        sa.UniqueConstraint("email", name=op.f("uq_users_email")),
    )

    # --- doctor_profiles -----------------------------------------------
    # One-to-one with a doctor account; pending until an admin verifies.
    op.create_table(
        "doctor_profiles",
        sa.Column("user_id", postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column("specialization", sa.Text(), nullable=False),
        sa.Column("license_number", sa.Text(), nullable=False),
        sa.Column(
            "license_status", license_status, server_default="pending", nullable=False
        ),
        sa.Column(
            "languages",
            postgresql.ARRAY(sa.Text()),
            server_default=sa.text("'{}'"),
            nullable=False,
        ),
        sa.Column(
            "experience_years", sa.Integer(), server_default=sa.text("0"), nullable=False
        ),
        sa.Column(
            "is_available", sa.Boolean(), server_default=sa.false(), nullable=False
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.CheckConstraint(
            "experience_years >= 0", name="ck_doctor_profiles_experience_non_negative"
        ),
        sa.ForeignKeyConstraint(
            ["user_id"],
            ["users.id"],
            name=op.f("fk_doctor_profiles_user_id_users"),
            ondelete="CASCADE",
        ),
        sa.PrimaryKeyConstraint("user_id", name=op.f("pk_doctor_profiles")),
        sa.UniqueConstraint(
            "license_number", name=op.f("uq_doctor_profiles_license_number")
        ),
    )
    # The matching pipeline only ever scans approved, available doctors.
    op.create_index(
        "idx_doctor_profiles_matchable",
        "doctor_profiles",
        ["specialization"],
        unique=False,
        postgresql_where=sa.text("is_available AND license_status = 'approved'"),
    )

    # --- articles ------------------------------------------------------
    # Curated first-aid content; grounds the triage pipeline from Week 11.
    op.create_table(
        "articles",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            server_default=sa.text("gen_random_uuid()"),
            nullable=False,
        ),
        sa.Column("title", sa.Text(), nullable=False),
        sa.Column("slug", sa.Text(), nullable=False),
        sa.Column("content", sa.Text(), nullable=False),
        # Traceability: guidance must always point back to a real source.
        sa.Column("source_name", sa.Text(), nullable=False),
        sa.Column("source_url", sa.Text(), nullable=True),
        sa.Column("category", sa.Text(), nullable=True),
        sa.Column("language", sa.Text(), server_default="en", nullable=False),
        sa.Column("published", sa.Boolean(), server_default=sa.false(), nullable=False),
        sa.Column("created_by", postgresql.UUID(as_uuid=True), nullable=True),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.ForeignKeyConstraint(
            ["created_by"],
            ["users.id"],
            name=op.f("fk_articles_created_by_users"),
            ondelete="SET NULL",
        ),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_articles")),
        sa.UniqueConstraint("slug", name=op.f("uq_articles_slug")),
    )
    op.create_index(
        "idx_articles_published",
        "articles",
        ["category", "language"],
        unique=False,
        postgresql_where=sa.text("published"),
    )

    # --- consultations -------------------------------------------------
    # Emergency sessions; scheduled visits live in appointments.
    op.create_table(
        "consultations",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            server_default=sa.text("gen_random_uuid()"),
            nullable=False,
        ),
        sa.Column("patient_id", postgresql.UUID(as_uuid=True), nullable=False),
        # Assigned by matching after creation, so nullable until then.
        sa.Column("doctor_id", postgresql.UUID(as_uuid=True), nullable=True),
        sa.Column(
            "status",
            consultation_status,
            server_default="requested",
            nullable=False,
        ),
        sa.Column("call_link", sa.Text(), nullable=True),
        sa.Column("started_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("ended_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.ForeignKeyConstraint(
            ["patient_id"],
            ["users.id"],
            name=op.f("fk_consultations_patient_id_users"),
            ondelete="CASCADE",
        ),
        sa.ForeignKeyConstraint(
            ["doctor_id"],
            ["users.id"],
            name=op.f("fk_consultations_doctor_id_users"),
            ondelete="SET NULL",
        ),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_consultations")),
    )
    op.create_index(
        "idx_consultations_patient",
        "consultations",
        ["patient_id", sa.text("created_at DESC")],
        unique=False,
    )
    op.create_index(
        "idx_consultations_doctor",
        "consultations",
        ["doctor_id", sa.text("created_at DESC")],
        unique=False,
    )

    # --- triage_logs ---------------------------------------------------
    # One row per AI triage interaction; half of conversational memory.
    op.create_table(
        "triage_logs",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            server_default=sa.text("gen_random_uuid()"),
            nullable=False,
        ),
        sa.Column("patient_id", postgresql.UUID(as_uuid=True), nullable=False),
        # The patient's free-text description, kept verbatim for audit.
        sa.Column("input_text", sa.Text(), nullable=False),
        # Shape evolves with the extractor, hence JSONB rather than columns.
        sa.Column("symptoms", postgresql.JSONB(), nullable=True),
        sa.Column("emergency_level", emergency_level, nullable=True),
        # Model confidence as a percentage, e.g. 94.00.
        sa.Column("confidence", sa.Numeric(precision=5, scale=2), nullable=True),
        sa.Column(
            "requires_doctor", sa.Boolean(), server_default=sa.false(), nullable=False
        ),
        sa.Column("suggested_action", sa.Text(), nullable=True),
        # The safety-validated response actually shown to the patient.
        sa.Column("response", sa.Text(), nullable=True),
        sa.Column(
            "escalated_to_consultation_id",
            postgresql.UUID(as_uuid=True),
            nullable=True,
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.CheckConstraint(
            "confidence IS NULL OR (confidence >= 0 AND confidence <= 100)",
            name="ck_triage_logs_confidence_range",
        ),
        sa.ForeignKeyConstraint(
            ["patient_id"],
            ["users.id"],
            name=op.f("fk_triage_logs_patient_id_users"),
            ondelete="CASCADE",
        ),
        sa.ForeignKeyConstraint(
            ["escalated_to_consultation_id"],
            ["consultations.id"],
            name=op.f("fk_triage_logs_escalated_to_consultation_id_consultations"),
            ondelete="SET NULL",
        ),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_triage_logs")),
    )
    # Conversational memory reads a patient's recent history, newest first.
    op.create_index(
        "idx_triage_logs_patient",
        "triage_logs",
        ["patient_id", sa.text("created_at DESC")],
        unique=False,
    )

    # --- symptom_reports -----------------------------------------------
    # The raw symptom series feeding conversational memory.
    op.create_table(
        "symptom_reports",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            server_default=sa.text("gen_random_uuid()"),
            nullable=False,
        ),
        sa.Column("patient_id", postgresql.UUID(as_uuid=True), nullable=False),
        # Offline-synced reports may not belong to a triage session.
        sa.Column("triage_log_id", postgresql.UUID(as_uuid=True), nullable=True),
        sa.Column("symptom", sa.Text(), nullable=False),
        sa.Column("description", sa.Text(), nullable=True),
        sa.Column(
            "reported_offline", sa.Boolean(), server_default=sa.false(), nullable=False
        ),
        # When the patient says it started, not when it was recorded.
        sa.Column("onset_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.ForeignKeyConstraint(
            ["patient_id"],
            ["users.id"],
            name=op.f("fk_symptom_reports_patient_id_users"),
            ondelete="CASCADE",
        ),
        sa.ForeignKeyConstraint(
            ["triage_log_id"],
            ["triage_logs.id"],
            name=op.f("fk_symptom_reports_triage_log_id_triage_logs"),
            ondelete="SET NULL",
        ),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_symptom_reports")),
    )
    op.create_index(
        "idx_symptom_reports_patient",
        "symptom_reports",
        ["patient_id", sa.text("created_at DESC")],
        unique=False,
    )

    # --- appointments --------------------------------------------------
    op.create_table(
        "appointments",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            server_default=sa.text("gen_random_uuid()"),
            nullable=False,
        ),
        sa.Column("patient_id", postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column("doctor_id", postgresql.UUID(as_uuid=True), nullable=True),
        sa.Column("scheduled_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column(
            "status", appointment_status, server_default="scheduled", nullable=False
        ),
        sa.Column("notes", sa.Text(), nullable=True),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.ForeignKeyConstraint(
            ["patient_id"],
            ["users.id"],
            name=op.f("fk_appointments_patient_id_users"),
            ondelete="CASCADE",
        ),
        sa.ForeignKeyConstraint(
            ["doctor_id"],
            ["users.id"],
            name=op.f("fk_appointments_doctor_id_users"),
            ondelete="SET NULL",
        ),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_appointments")),
    )
    op.create_index(
        "idx_appointments_patient",
        "appointments",
        ["patient_id", sa.text("scheduled_at DESC")],
        unique=False,
    )
    op.create_index(
        "idx_appointments_doctor",
        "appointments",
        ["doctor_id", sa.text("scheduled_at DESC")],
        unique=False,
    )

    # --- medical_history -----------------------------------------------
    op.create_table(
        "medical_history",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            server_default=sa.text("gen_random_uuid()"),
            nullable=False,
        ),
        sa.Column("patient_id", postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column("condition", sa.Text(), nullable=False),
        sa.Column("notes", sa.Text(), nullable=True),
        sa.Column("diagnosed_on", sa.Date(), nullable=True),
        # Doctor or admin who entered the record, when not self-reported.
        sa.Column("recorded_by", postgresql.UUID(as_uuid=True), nullable=True),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.ForeignKeyConstraint(
            ["patient_id"],
            ["users.id"],
            name=op.f("fk_medical_history_patient_id_users"),
            ondelete="CASCADE",
        ),
        sa.ForeignKeyConstraint(
            ["recorded_by"],
            ["users.id"],
            name=op.f("fk_medical_history_recorded_by_users"),
            ondelete="SET NULL",
        ),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_medical_history")),
    )
    op.create_index(
        "idx_medical_history_patient", "medical_history", ["patient_id"], unique=False
    )

    # --- doctor_ratings ------------------------------------------------
    # doctor_id duplicated from the consultation so per-doctor aggregation
    # needs no join on the hot path.
    op.create_table(
        "doctor_ratings",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            server_default=sa.text("gen_random_uuid()"),
            nullable=False,
        ),
        sa.Column("consultation_id", postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column("doctor_id", postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column("rating", sa.SmallInteger(), nullable=False),
        sa.Column("feedback", sa.Text(), nullable=True),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.CheckConstraint(
            "rating BETWEEN 1 AND 5", name="ck_doctor_ratings_rating_range"
        ),
        sa.ForeignKeyConstraint(
            ["consultation_id"],
            ["consultations.id"],
            name=op.f("fk_doctor_ratings_consultation_id_consultations"),
            ondelete="CASCADE",
        ),
        sa.ForeignKeyConstraint(
            ["doctor_id"],
            ["users.id"],
            name=op.f("fk_doctor_ratings_doctor_id_users"),
            ondelete="CASCADE",
        ),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_doctor_ratings")),
        sa.UniqueConstraint(
            "consultation_id", name=op.f("uq_doctor_ratings_consultation_id")
        ),
    )
    op.create_index(
        "idx_doctor_ratings_doctor", "doctor_ratings", ["doctor_id"], unique=False
    )

    # --- notifications -------------------------------------------------
    op.create_table(
        "notifications",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            server_default=sa.text("gen_random_uuid()"),
            nullable=False,
        ),
        sa.Column("user_id", postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column("channel", notification_channel, nullable=False),
        sa.Column("event_type", sa.Text(), nullable=False),
        sa.Column("payload", postgresql.JSONB(), nullable=True),
        sa.Column(
            "status", notification_status, server_default="pending", nullable=False
        ),
        sa.Column("sent_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.ForeignKeyConstraint(
            ["user_id"],
            ["users.id"],
            name=op.f("fk_notifications_user_id_users"),
            ondelete="CASCADE",
        ),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_notifications")),
    )
    op.create_index(
        "idx_notifications_user",
        "notifications",
        ["user_id", sa.text("created_at DESC")],
        unique=False,
    )
    # The dispatcher polls only undelivered rows.
    op.create_index(
        "idx_notifications_pending",
        "notifications",
        ["created_at"],
        unique=False,
        postgresql_where=sa.text("status = 'pending'"),
    )

    # --- emergency_contacts --------------------------------------------
    op.create_table(
        "emergency_contacts",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            server_default=sa.text("gen_random_uuid()"),
            nullable=False,
        ),
        sa.Column("patient_id", postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column("name", sa.Text(), nullable=False),
        sa.Column("phone", sa.Text(), nullable=False),
        sa.Column("relationship", sa.Text(), nullable=True),
        # Lower numbers are called first.
        sa.Column(
            "sort_order", sa.SmallInteger(), server_default=sa.text("1"), nullable=False
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.ForeignKeyConstraint(
            ["patient_id"],
            ["users.id"],
            name=op.f("fk_emergency_contacts_patient_id_users"),
            ondelete="CASCADE",
        ),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_emergency_contacts")),
    )
    op.create_index(
        "idx_emergency_contacts_patient",
        "emergency_contacts",
        ["patient_id", "sort_order"],
        unique=False,
    )


def downgrade() -> None:
    # Reverse creation order: dependents before the tables they reference.
    for table in (
        "emergency_contacts",
        "notifications",
        "doctor_ratings",
        "medical_history",
        "appointments",
        "symptom_reports",
        "triage_logs",
        "consultations",
        "articles",
        "doctor_profiles",
        "users",
    ):
        op.drop_table(table)

    bind = op.get_bind()
    for enum_type in reversed(ENUM_TYPES):
        enum_type.drop(bind, checkfirst=True)
