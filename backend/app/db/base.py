"""The metadata Alembic autogenerates against.

Importing this module imports every table model exactly once, so
``SQLModel.metadata`` is complete. Alembic's ``env.py`` and the test fixtures
import from here rather than reaching into ``app.models`` directly — a model
that is not reachable from this module will be silently dropped from
generated migrations.
"""

from sqlmodel import SQLModel

# Re-exported for `target_metadata` in alembic/env.py.
from app.models import (
    Appointment,
    Article,
    Consultation,
    DoctorProfile,
    DoctorRating,
    EmergencyContact,
    MedicalHistory,
    Notification,
    SymptomReport,
    TriageLog,
    User,
)

metadata = SQLModel.metadata

__all__ = ["SQLModel", "metadata"]
