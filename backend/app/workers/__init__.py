"""Celery workers -- everything that must not block a request.

Two queues, deliberately separated in ``app.core.celery_app``:

``ai``
    Model inference, transcription, and speech synthesis. Slow (seconds to
    tens of seconds) and bursty.
``notifications``
    Email, SMS, and push delivery. Fast, but must not queue behind a
    backlog of model calls -- a Red-level emergency alert waiting on
    somebody else's transcription would be a product failure, so the split
    is a safety property rather than a performance tweak.
"""
