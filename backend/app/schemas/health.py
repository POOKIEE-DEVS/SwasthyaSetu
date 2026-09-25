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
