"""Database engine and sessions.

Postgres in production; SQLite for local development and tests. Tables are
created at startup with ``SQLModel.metadata.create_all``, which only adds
missing tables. That suits the demo's additive schema; move to Alembic
migrations before changing existing columns on a database with real data.

Sessions are synchronous. FastAPI runs sync endpoints and dependencies in a
worker thread, and async code (chat, WebSockets) calls the database through
``run_in_threadpool``, so queries never block the event loop that carries
call signalling.
"""

from __future__ import annotations

from collections.abc import Iterator

from sqlalchemy import Engine, event
from sqlalchemy.pool import StaticPool
from sqlmodel import Session, SQLModel, create_engine

from app.core.config import settings

_engine: Engine | None = None


def make_engine(url: str) -> Engine:
    if url.startswith("sqlite"):
        in_memory = url in ("sqlite://", "sqlite:///:memory:")
        engine = create_engine(
            url,
            connect_args={"check_same_thread": False},
            # One shared connection, so every session sees the same
            # in-memory database.
            poolclass=StaticPool if in_memory else None,
        )

        @event.listens_for(engine, "connect")
        def _enable_foreign_keys(dbapi_connection, _record) -> None:
            cursor = dbapi_connection.cursor()
            cursor.execute("PRAGMA foreign_keys=ON")
            cursor.close()

        return engine

    # Hosted Postgres (Neon, Supabase) closes idle connections; pre-ping
    # replaces a dead one instead of failing the request.
    return create_engine(
        url, pool_pre_ping=True, pool_size=5, max_overflow=5, pool_recycle=300
    )


def get_engine() -> Engine:
    global _engine
    if _engine is None:
        _engine = make_engine(settings.sqlalchemy_database_url)
    return _engine


def use_engine(engine: Engine) -> None:
    """Swap the engine (tests use an in-memory database)."""
    global _engine
    _engine = engine


def init_db() -> None:
    from app import models  # noqa: F401  (registers the tables)

    SQLModel.metadata.create_all(get_engine())


def get_session() -> Iterator[Session]:
    with Session(get_engine()) as session:
        yield session
