"""Wire format of the health and readiness endpoints."""

from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field


class HealthResponse(BaseModel):
    """Liveness: the process is up and serving.

    Deliberately depends on nothing external -- an orchestrator restarting
    the container because Postgres blipped would turn a recoverable outage
    into an unrecoverable one.
    """

    status: Literal["ok"] = "ok"
    service: str
    version: str
    uptime_seconds: float


class DependencyStatus(BaseModel):
    """Result of probing one backing service."""

    status: Literal["ok", "error"]
    latency_ms: float | None = None
    detail: str | None = Field(
        default=None, description="Error class and message when status is error."
    )


class ReadinessResponse(BaseModel):
    """Readiness: every dependency needed to serve traffic is reachable.

    Returned with HTTP 503 when any dependency is down, so a load balancer
    stops routing to this instance without the process being killed.
    """

    status: Literal["ready", "degraded"]
    service: str
    version: str
    uptime_seconds: float
    dependencies: dict[str, DependencyStatus]
