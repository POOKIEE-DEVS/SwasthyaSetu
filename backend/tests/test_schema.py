"""Guards on the model registry.

Alembic autogenerates against ``app.db.base.metadata``. A model that stops
being reachable from there disappears from generated migrations silently, so
the count and the names are asserted rather than assumed.
"""

from __future__ import annotations

import pytest

from app.db.base import metadata

# The eleven tables of architecture section 6.
EXPECTED_TABLES = {
    "appointments",
    "articles",
    "consultations",
    "doctor_profiles",
    "doctor_ratings",
    "emergency_contacts",
    "medical_history",
    "notifications",
    "symptom_reports",
    "triage_logs",
    "users",
}


def test_every_table_is_registered() -> None:
    assert set(metadata.tables) == EXPECTED_TABLES


@pytest.mark.parametrize(
    ("table", "index"),
    [
        ("doctor_profiles", "idx_doctor_profiles_matchable"),
        ("articles", "idx_articles_published"),
        ("notifications", "idx_notifications_pending"),
    ],
)
def test_partial_indexes_survive(table: str, index: str) -> None:
    """These carry a WHERE clause; losing it silently would be a full scan."""
    target = next(i for i in metadata.tables[table].indexes if i.name == index)
    assert target.dialect_options["postgresql"]["where"] is not None


def test_patient_data_cascades_from_users() -> None:
    """Deleting an account must not strand that person's medical records."""
    for table_name, column in [
        ("triage_logs", "patient_id"),
        ("symptom_reports", "patient_id"),
        ("medical_history", "patient_id"),
        ("emergency_contacts", "patient_id"),
        ("consultations", "patient_id"),
    ]:
        fk = next(iter(metadata.tables[table_name].c[column].foreign_keys))
        assert fk.ondelete == "CASCADE", f"{table_name}.{column}"
