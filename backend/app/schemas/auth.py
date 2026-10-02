from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field

from app.schemas.verification import VerificationSummary

Role = Literal["patient", "doctor", "pharmacist", "student"]


class UserPublic(BaseModel):
    id: int
    email: str
    name: str
    picture_url: str | None
    role: Role | None
    is_admin: bool
    # Professionals only: where their verification stands.
    verification: VerificationSummary | None = None


class MeResponse(BaseModel):
    # None when signed out. Always 200, so the page can check quietly.
    user: UserPublic | None
    google_enabled: bool
    dev_login: bool


class RoleChoice(BaseModel):
    role: Role


class DevLoginRequest(BaseModel):
    email: str = Field(pattern=r"^[^@\s]+@[^@\s]+\.[^@\s]+$", max_length=320)
    name: str = Field(min_length=1, max_length=120)
