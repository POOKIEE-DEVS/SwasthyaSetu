"""Shared FastAPI dependencies.

Re-exports the database and cache dependencies under stable annotated
aliases, so routers depend on ``SessionDep`` rather than reaching into
``app.db`` and each spelling ``Depends(...)`` themselves.

Authentication dependencies (``CurrentUser``, ``require_role``) arrive in
Weeks 3-4 and belong here too.
"""

from __future__ import annotations

from typing import Annotated

from fastapi import Depends
from redis.asyncio import Redis
from sqlalchemy.ext.asyncio import AsyncSession

from app.db.redis import get_redis
from app.db.session import get_session

SessionDep = Annotated[AsyncSession, Depends(get_session)]
RedisDep = Annotated[Redis, Depends(get_redis)]

__all__ = ["RedisDep", "SessionDep"]
