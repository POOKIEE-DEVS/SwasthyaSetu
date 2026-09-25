"""Liveness and readiness endpoints.

The split matters operationally: ``/health`` answers "is this process
alive?" and must never depend on Postgres or Redis, while ``/health/ready``
answers "should traffic be routed here?" and probes both.
"""

from __future__ import annotations

import logging
import time

from fastapi import APIRouter, Response, status
from sqlalchemy import text

from app import __version__
from app.api.deps import RedisDep, SessionDep
from app.core.config import settings
from app.schemas.health import DependencyStatus, HealthResponse, ReadinessResponse

logger = logging.getLogger(__name__)
router = APIRouter(tags=["health"])

_started_at = time.monotonic()


def _uptime_seconds() -> float:
    return round(time.monotonic() - _started_at, 3)


@router.get("/health", response_model=HealthResponse, summary="Liveness probe")
async def health() -> HealthResponse:
    return HealthResponse(
        service=settings.service_name,
        version=__version__,
        uptime_seconds=_uptime_seconds(),
    )


async def _probe(name: str, coro) -> DependencyStatus:
    """Run one dependency check, timing it and converting failure to a status.

    A readiness probe that raises is useless -- the caller needs the report,
    so every failure becomes data rather than a 500.
    """
    started = time.perf_counter()
    try:
        await coro
    except Exception as exc:
        logger.warning("readiness: %s check failed", name, exc_info=exc)
        return DependencyStatus(
            status="error",
            latency_ms=round((time.perf_counter() - started) * 1000, 2),
            detail=f"{type(exc).__name__}: {exc}",
        )
    return DependencyStatus(
        status="ok",
        latency_ms=round((time.perf_counter() - started) * 1000, 2),
    )


@router.get(
    "/health/ready",
    response_model=ReadinessResponse,
    summary="Readiness probe",
    responses={503: {"description": "One or more dependencies are unreachable."}},
)
async def readiness(
    session: SessionDep, redis: RedisDep, response: Response
) -> ReadinessResponse:
    dependencies = {
        "postgres": await _probe("postgres", session.execute(text("SELECT 1"))),
        "redis": await _probe("redis", redis.ping()),
    }

    degraded = any(dep.status == "error" for dep in dependencies.values())
    if degraded:
        response.status_code = status.HTTP_503_SERVICE_UNAVAILABLE

    return ReadinessResponse(
        status="degraded" if degraded else "ready",
        service=settings.service_name,
        version=__version__,
        uptime_seconds=_uptime_seconds(),
        dependencies=dependencies,
    )
