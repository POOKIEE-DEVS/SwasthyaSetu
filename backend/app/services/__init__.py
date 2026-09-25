"""Shared in-process state for the demo: consultations and rate limits."""

from app.core.config import settings
from app.services.consultations import ConsultationRegistry
from app.services.rate_limit import RateLimiter

consultations = ConsultationRegistry(ttl_seconds=settings.consultation_ttl_minutes * 60)
chat_rate_limiter = RateLimiter(limit=settings.chat_rate_limit_per_minute)
