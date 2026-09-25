"""Celery application — background and long-running work.

Anything that must not block an HTTP request or a WebSocket frame runs here:
model inference, speech-to-text and text-to-speech, notification delivery,
and offline-sync reconciliation. Redis is both broker and result backend.

Run a worker with::

    celery -A app.core.celery_app:celery_app worker --loglevel=info
"""

from __future__ import annotations

from celery import Celery

from app.core.config import settings

celery_app = Celery(
    "swasthyasetu",
    broker=settings.celery_broker_url,
    backend=settings.celery_result_backend,
    include=["app.workers.tasks"],
)

celery_app.conf.update(
    task_serializer="json",
    result_serializer="json",
    accept_content=["json"],
    timezone="UTC",
    enable_utc=True,
    # Fetch one message at a time: tasks here are long and uneven (a model
    # inference vs. an SMS send), so prefetching would leave work stranded
    # behind a slow task on one worker while others idle.
    worker_prefetch_multiplier=1,
    task_acks_late=True,
    task_reject_on_worker_lost=True,
    task_soft_time_limit=settings.celery_task_soft_time_limit,
    task_time_limit=settings.celery_task_time_limit,
    result_expires=3600,
    # Route heavy ML work away from short transactional tasks so a queue of
    # inferences can never delay an emergency notification.
    task_routes={
        "ai.*": {"queue": "ai"},
        "notifications.*": {"queue": "notifications"},
    },
    task_default_queue="default",
)
