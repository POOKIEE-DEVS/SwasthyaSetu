"""SQLModel table models -- the eleven tables of architecture section 6.

Import models from this package (never from the submodules directly) so that
every table is registered on ``SQLModel.metadata`` exactly once.
"""

from app.models.consultation import Appointment, Consultation
from app.models.content import Article
from app.models.doctor import DoctorProfile, DoctorRating
from app.models.enums import (
    AppointmentStatus,
    ConsultationStatus,
    EmergencyLevel,
    LicenseStatus,
    NotificationChannel,
    NotificationStatus,
    UserRole,
)
from app.models.notification import Notification
from app.models.patient import EmergencyContact, MedicalHistory
from app.models.triage import SymptomReport, TriageLog
from app.models.user import User

__all__ = [
    # Tables
    "Appointment",
    "Article",
    "Consultation",
    "DoctorProfile",
    "DoctorRating",
    "EmergencyContact",
    "MedicalHistory",
    "Notification",
    "SymptomReport",
    "TriageLog",
    "User",
    # Enums
    "AppointmentStatus",
    "ConsultationStatus",
    "EmergencyLevel",
    "LicenseStatus",
    "NotificationChannel",
    "NotificationStatus",
    "UserRole",
]
