"""Pydantic request and response schemas -- the shapes on the wire.

Kept separate from ``app.models`` (which is persistence) so that the API
contract can evolve without dragging the database schema with it, and so a
column is never accidentally exposed by being added to a table.
"""

from app.schemas.health import DependencyStatus, HealthResponse, ReadinessResponse

__all__ = ["DependencyStatus", "HealthResponse", "ReadinessResponse"]
