"""Redis client — cache, sessions, and rate-limit counters.

Celery uses its own connection to the broker/result databases; this client is
for the application's own reads and writes on the cache database
(``REDIS_URL``).
"""

from __future__ import annotations

from redis.asyncio import ConnectionPool, Redis

from app.core.config import settings

_pool = ConnectionPool.from_url(
    settings.redis_url,
    max_connections=settings.redis_max_connections,
    decode_responses=True,
)

redis_client: Redis = Redis(connection_pool=_pool)


async def get_redis() -> Redis:
    """FastAPI dependency returning the shared client.

    The client is connection-pooled and safe to share; there is deliberately
    no per-request client to create and tear down.
    """
    return redis_client


async def close_redis() -> None:
    """Release pooled connections. Called from the lifespan shutdown."""
    await redis_client.aclose()
    await _pool.aclose()
