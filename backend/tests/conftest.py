"""Shared fixtures.

These tests run without Postgres or Redis: dependencies are overridden with
fakes so the suite stays fast and CI needs no service containers. Tests that
exercise real SQL arrive in Week 2 alongside the seed script.
"""

from __future__ import annotations

from collections.abc import Iterator

import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from app.api.deps import get_redis, get_session
from app.main import create_app


class FakeResult:
    """Stand-in for the object `AsyncSession.execute` resolves to."""

    def scalar(self) -> int:
        return 1


class FakeSession:
    """Minimal async session: readiness only ever runs `SELECT 1`."""

    def __init__(self, *, fail: bool = False) -> None:
        self._fail = fail

    async def execute(self, *_args, **_kwargs) -> FakeResult:
        if self._fail:
            raise ConnectionRefusedError("connection to postgres refused")
        return FakeResult()


class FakeRedis:
    def __init__(self, *, fail: bool = False) -> None:
        self._fail = fail

    async def ping(self) -> bool:
        if self._fail:
            raise ConnectionError("connection to redis refused")
        return True


def build_app(*, postgres_fails: bool = False, redis_fails: bool = False) -> FastAPI:
    """An app whose backing services are fakes with controllable failure."""
    app = create_app()

    async def _session():
        yield FakeSession(fail=postgres_fails)

    async def _redis():
        return FakeRedis(fail=redis_fails)

    app.dependency_overrides[get_session] = _session
    app.dependency_overrides[get_redis] = _redis
    return app


@pytest.fixture
def client() -> Iterator[TestClient]:
    """Client for an app whose dependencies are all healthy."""
    with TestClient(build_app()) as test_client:
        yield test_client
