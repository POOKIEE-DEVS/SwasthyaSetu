"""SwasthyaSetu backend: app assembly.

One process serves everything from a single origin:

- ``/api/v1/*``  REST API (chat, consultations, health)
- ``/ws/*``      WebSockets (doctor queue, call signalling)
- ``/``          the Next.js static export, when ``STATIC_DIR`` exists

The same origin means no CORS and no build-time API URL in production. Run it
as a single worker (``uvicorn app.main:app``, no ``--workers``). Consultations
and WebSocket rooms live in this process's memory, so a second worker would
split them and calls would fail to connect.
"""

from __future__ import annotations

import logging
import mimetypes
import time
import uuid
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles

from app import __version__
from app.api import api_router
from app.api import health as health_routes
from app.core.config import settings
from app.core.logging import configure_logging
from app.realtime import websocket as websocket_routes

logger = logging.getLogger(__name__)

# Not in Python's default table; without it the manifest is served as
# application/octet-stream and browsers ignore it.
mimetypes.add_type("application/manifest+json", ".webmanifest")


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    configure_logging(settings.log_level, json_output=settings.log_json)
    logger.info(
        "backend starting",
        extra={
            "version": __version__,
            "environment": settings.environment,
            "model_configured": settings.model_configured,
            "serving_frontend": settings.static_path.is_dir(),
        },
    )
    yield


def create_app() -> FastAPI:
    app = FastAPI(
        title="SwasthyaSetu API",
        version=__version__,
        lifespan=lifespan,
        docs_url="/api/docs",
        redoc_url=None,
        openapi_url="/api/openapi.json",
    )

    if settings.cors_origin_list:
        app.add_middleware(
            CORSMiddleware,
            allow_origins=settings.cors_origin_list,
            allow_credentials=False,
            allow_methods=["*"],
            allow_headers=["*"],
        )

    @app.middleware("http")
    async def request_log(request: Request, call_next):
        request_id = request.headers.get("X-Request-ID") or uuid.uuid4().hex
        started = time.perf_counter()
        response = await call_next(request)
        response.headers["X-Request-ID"] = request_id
        if request.url.path.startswith("/api/"):
            logger.info(
                "request",
                extra={
                    "request_id": request_id,
                    "method": request.method,
                    "path": request.url.path,
                    "status_code": response.status_code,
                    "duration_ms": round((time.perf_counter() - started) * 1000, 1),
                },
            )
        return response

    # /health at the root too, for the hosting platform's health check.
    app.include_router(health_routes.router)
    app.include_router(api_router, prefix="/api/v1")
    app.include_router(websocket_routes.router)

    # Mounted last, so every API and WebSocket route above takes precedence.
    if settings.static_path.is_dir():
        app.mount(
            "/", StaticFiles(directory=settings.static_path, html=True), name="web"
        )

    return app


app = create_app()
