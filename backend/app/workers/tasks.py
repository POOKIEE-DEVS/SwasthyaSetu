"""Celery task definitions.

Task names are explicit and namespaced (``ai.*``, ``notifications.*``)
because ``task_routes`` in ``app.core.celery_app`` routes on them. A task
that is renamed without updating that mapping silently lands on the default
queue, so the names are part of the contract, not incidental.

Planned tasks:

``ai.run_triage``            Week 9-12   Full six-stage pipeline for one
                                         symptom description; writes a
                                         ``triage_logs`` row.
``ai.transcribe_audio``      Week 10     whisper.cpp on an uploaded Nepali
                                         voice note.
``ai.synthesize_speech``     Week 12     Nepali narration of validated
                                         guidance, cached by content hash.
``notifications.dispatch``   Week 14     Deliver one queued notification
                                         over email / SMS / push.
``notifications.sweep``      Week 14     Periodic retry of rows still
                                         ``pending`` past their deadline.
"""

from __future__ import annotations

import logging
import time

from app.core.celery_app import celery_app

logger = logging.getLogger(__name__)


@celery_app.task(name="ops.ping")
def ping() -> dict[str, float | str]:
    """Round-trip check that a worker is consuming from the broker.

    Exists from Week 1 so the worker container is verifiably doing
    something before any real task is written, and stays afterwards as the
    cheapest possible "is the queue moving?" probe.
    """
    logger.info("ping task executed")
    return {"status": "ok", "worker_time": time.time()}
