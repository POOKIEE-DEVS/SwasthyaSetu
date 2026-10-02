from __future__ import annotations

from typing import Literal

from pydantic import BaseModel


class HealthResponse(BaseModel):
    status: Literal["ok"] = "ok"
    version: str
    uptime_seconds: float
    # Configuration presence only. Neither is probed live, because probing
    # would wake a sleeping GPU Space on every health check.
    model_configured: bool
    turn_configured: bool
    # "sqlite" in production means DATABASE_URL is missing: accounts and
    # approvals would be lost on the next restart.
    database: Literal["postgres", "sqlite"]
    google_sign_in_configured: bool
