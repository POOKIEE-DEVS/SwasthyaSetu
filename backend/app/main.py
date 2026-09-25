"""SwasthyaSetu backend -- application entry point.

Only app assembly lives here: configuration, middleware, lifespan, and
router mounting. Domain logic belongs in ``app.services``, the triage
pipeline in ``app.ai``, and persistence in ``app.models`` / ``app.db``.

Run locally with::

    uvicorn app.main:app --reload

and in production behind Gunicorn's process manager::

    gunicorn app.main:app -k uvicorn.workers.UvicornWorker -w 4
"""

from __future__ import annotations

import logging
import time
import uuid
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from starlette.middleware.gzip import GZipMiddleware

from app import __version__
from app.api.v1 import health as health_routes
from app.api.v1.router import api_router
from app.core.config import settings
from app.core.logging import configure_logging
from app.db.redis import close_redis
from app.db.session import dispose_engine
from app.realtime import websocket as websocket_routes

logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    """Start-up and shut-down hooks.

    Connections are opened lazily by their pools rather than eagerly here:
    the backend must still start and serve ``/health`` when Postgres or
    Redis is briefly unavailable, so that a dependency outage shows up as a
    readiness failure instead of a crash loop.
    """
    configure_logging(settings.log_level, json_output=settings.log_json)
    logger.info(
        "backend starting",
        extra={
            "service": settings.service_name,
            "version": __version__,
            "environment": settings.environment,
        },
    )

    yield

    logger.info("backend shutting down")
    await dispose_engine()
    await close_redis()


def create_app() -> FastAPI:
    app = FastAPI(
        title="SwasthyaSetu API",
        description=(
            "Healthcare access platform: authentication, doctor matching, "
            "consultations, notifications, and the AI triage pipeline."
        ),
        version=__version__,
        lifespan=lifespan,
        docs_url=None if settings.is_production else "/docs",
        redoc_url=None if settings.is_production else "/redoc",
        openapi_url=None if settings.is_production else "/openapi.json",
    )

    app.add_middleware(
        CORSMiddleware,
        allow_origins=settings.cors_origins,
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )
    # First-aid articles are large and highly compressible, and the target
    # users are on slow mobile connections.
    app.add_middleware(GZipMiddleware, minimum_size=1024)

    @app.middleware("http")
    async def request_context(request: Request, call_next):
        """Tag every request with an id and log its outcome and duration.

        The id is echoed as ``X-Request-ID`` so a user-reported failure can
        be traced to exact log lines (architecture section 12). An inbound
        header is honoured when present, which keeps the id stable across
        the frontend-to-backend hop.
        """
        request_id = request.headers.get("X-Request-ID") or uuid.uuid4().hex
        request.state.request_id = request_id
        started = time.perf_counter()

        try:
            response = await call_next(request)
        except Exception:
            logger.exception(
                "request failed",
                extra={
                    "request_id": request_id,
                    "method": request.method,
                    "path": request.url.path,
                    "duration_ms": round((time.perf_counter() - started) * 1000, 2),
                },
            )
            raise

        duration_ms = round((time.perf_counter() - started) * 1000, 2)
        response.headers["X-Request-ID"] = request_id
        logger.info(
            "request completed",
            extra={
                "request_id": request_id,
                "method": request.method,
                "path": request.url.path,
                "status_code": response.status_code,
                "duration_ms": duration_ms,
            },
        )
        return response

    # Probes are mounted at the root as well as under the version prefix:
    # orchestrators and load balancers should not have to know about API
    # versioning to check whether a container is alive.
    app.include_router(health_routes.router)
    app.include_router(api_router, prefix=settings.api_v1_prefix)
    app.include_router(websocket_routes.router)

    return app


app = create_app()
