from __future__ import annotations

import time

from fastapi import APIRouter

from app import __version__
from app.core.config import settings
from app.realtime.ice import turn_configured
from app.schemas.health import HealthResponse

router = APIRouter(tags=["health"])
_started_at = time.monotonic()


@router.get("/health", response_model=HealthResponse)
async def health() -> HealthResponse:
    return HealthResponse(
        version=__version__,
        uptime_seconds=round(time.monotonic() - _started_at, 3),
        model_configured=settings.model_configured,
        turn_configured=turn_configured(),
    )
