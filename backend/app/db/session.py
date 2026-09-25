"""SQLAlchemy engines and session factories.

The request path is fully async (asyncpg). Celery workers and Alembic run
outside it and use the blocking engine built lazily by
:func:`get_sync_session_factory` — see ``Settings.sync_database_url``.
"""

from __future__ import annotations

from collections.abc import AsyncGenerator
from functools import lru_cache

from sqlalchemy.ext.asyncio import (
    AsyncEngine,
    AsyncSession,
    async_sessionmaker,
    create_async_engine,
)
from sqlalchemy.orm import Session, sessionmaker

from app.core.config import settings

engine: AsyncEngine = create_async_engine(
    settings.database_url,
    echo=settings.db_echo,
    pool_size=settings.db_pool_size,
    max_overflow=settings.db_max_overflow,
    # Postgres and connection poolers drop idle connections; recycling below
    # their timeout avoids handing a dead connection to a request.
    pool_recycle=settings.db_pool_recycle_seconds,
    pool_pre_ping=True,
)

async_session_factory = async_sessionmaker(
    engine,
    class_=AsyncSession,
    expire_on_commit=False,  # let handlers read attributes after commit
    autoflush=False,
)


async def get_session() -> AsyncGenerator[AsyncSession, None]:
    """FastAPI dependency yielding a session scoped to one request.

    The session is rolled back and closed on any exception so a failed
    request can never leak a half-applied transaction into the pool.
    """
    async with async_session_factory() as session:
        try:
            yield session
        except Exception:
            await session.rollback()
            raise


@lru_cache(maxsize=1)
def get_sync_session_factory() -> sessionmaker[Session]:
    """Blocking session factory for Celery tasks.

    Built lazily and cached: the web process never touches it, so importing
    this module must not open a second pool of connections.
    """
    from sqlalchemy import create_engine

    sync_engine = create_engine(
        settings.sync_database_url,
        echo=settings.db_echo,
        pool_pre_ping=True,
    )
    return sessionmaker(sync_engine, expire_on_commit=False, autoflush=False)


async def dispose_engine() -> None:
    """Close the async pool. Called from the application lifespan shutdown."""
    await engine.dispose()
