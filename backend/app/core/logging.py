"""Structured logging setup.

Every service emits structured request, error, and performance logs so that
failures are debuggable in development and demonstrable in production
(architecture §12). Development gets human-readable lines; production gets
one JSON object per line for ingestion by a log aggregator.
"""

from __future__ import annotations

import json
import logging
import sys
from typing import Any

# Attributes present on every LogRecord. Anything else was attached by the
# caller via `extra=` and belongs in the structured output.
_RESERVED = frozenset(logging.LogRecord("", 0, "", 0, "", None, None).__dict__) | {
    "asctime",
    "message",
    "taskName",
}


class JsonFormatter(logging.Formatter):
    """Render a record as a single JSON object, including `extra` fields."""

    def format(self, record: logging.LogRecord) -> str:
        payload: dict[str, Any] = {
            "timestamp": self.formatTime(record, "%Y-%m-%dT%H:%M:%S%z"),
            "level": record.levelname,
            "logger": record.name,
            "message": record.getMessage(),
        }
        for key, value in record.__dict__.items():
            if key not in _RESERVED:
                payload[key] = value
        if record.exc_info:
            payload["exception"] = self.formatException(record.exc_info)
        return json.dumps(payload, default=str)


def configure_logging(level: str = "INFO", *, json_output: bool = False) -> None:
    """Install a single stdout handler on the root logger.

    Called once at application startup. Uvicorn installs its own handlers, so
    we clear existing ones first to avoid every line being emitted twice.
    """
    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(
        JsonFormatter()
        if json_output
        else logging.Formatter(
            "%(asctime)s %(levelname)-8s %(name)s: %(message)s",
            datefmt="%H:%M:%S",
        )
    )

    root = logging.getLogger()
    root.handlers.clear()
    root.addHandler(handler)
    root.setLevel(level.upper())

    # Let uvicorn's loggers propagate to root rather than formatting their own.
    for name in ("uvicorn", "uvicorn.error", "uvicorn.access"):
        logging.getLogger(name).handlers.clear()
        logging.getLogger(name).propagate = True
