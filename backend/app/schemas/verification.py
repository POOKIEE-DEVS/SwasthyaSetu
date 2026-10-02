from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field

ProfessionalRole = Literal["doctor", "pharmacist", "student"]
VerificationStatus = Literal["pending", "approved", "rejected"]
DocumentKind = Literal[
    "citizenship_front",
    "citizenship_back",
    "council_certificate",
    "recommendation_letter",
    "selfie",
]


class DocumentInfo(BaseModel):
    id: int
    kind: DocumentKind
    content_type: str
    size: int


class AuditPublic(BaseModel):
    action: Literal["submitted", "resubmitted", "approved", "rejected"]
    actor_email: str
    reason: str | None
    at: float


class ApplicationPublic(BaseModel):
    """What applicants see of their own application."""

    id: int
    role: ProfessionalRole
    full_name: str
    phone: str
    citizenship_number: str
    citizenship_district: str
    council_number: str | None
    institution: str | None
    recommender_name: str | None
    recommender_nmc: str | None
    status: VerificationStatus
    rejection_reason: str | None
    submitted_at: float
    reviewed_at: float | None
    documents: list[DocumentInfo]


class AdminApplication(ApplicationPublic):
    """The admin's review card: the application plus account and history."""

    user_email: str
    user_name: str
    user_picture_url: str | None
    history: list[AuditPublic]


class RejectRequest(BaseModel):
    # Shown to the applicant, so they know what to fix.
    reason: str = Field(min_length=3, max_length=500)


class VerificationSummary(BaseModel):
    """Embedded in /auth/me so every page knows the user's standing."""

    role: ProfessionalRole
    status: VerificationStatus
    rejection_reason: str | None
    full_name: str
