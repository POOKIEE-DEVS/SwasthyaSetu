"""Client for the MedGemma Gradio Space (see model-space/).

The Space exposes one API endpoint, ``/generate``. It takes the conversation
as a JSON string and returns the reply text. ``gradio_client`` handles the
Space's queue protocol and authentication. Its calls block, so they run in a
worker thread under a timeout and never stall the event loop, which also
carries WebRTC signalling.
"""

from __future__ import annotations

import asyncio
import json
import logging
import threading

from gradio_client import Client

from app.core.config import settings

logger = logging.getLogger(__name__)


class ModelUnavailableError(Exception):
    """The model could not produce a reply (not configured, asleep, erroring)."""


class MedGemmaClient:
    def __init__(self, space_id: str, token: str, timeout: float) -> None:
        self._space_id = space_id
        self._token = token or None
        self._timeout = timeout
        self._client: Client | None = None
        self._lock = threading.Lock()

    def _get_client(self) -> Client:
        # Connecting fetches the Space's API schema, and will wake a sleeping
        # Space, so the client is created once and reused.
        with self._lock:
            if self._client is None:
                self._client = Client(self._space_id, token=self._token, verbose=False)
            return self._client

    def _reset(self) -> None:
        with self._lock:
            self._client = None

    def _generate_sync(self, messages: list[dict[str, str]]) -> str:
        client = self._get_client()
        result = client.predict(json.dumps(messages), api_name="/generate")
        return str(result).strip()

    async def generate(self, messages: list[dict[str, str]]) -> str:
        if not self._space_id:
            raise ModelUnavailableError("The AI model is not configured.")
        try:
            reply = await asyncio.wait_for(
                asyncio.to_thread(self._generate_sync, messages),
                timeout=self._timeout,
            )
        except TimeoutError as exc:
            raise ModelUnavailableError(
                "The AI model took too long to answer. It may be waking up; "
                "please try again in a minute."
            ) from exc
        except Exception as exc:
            # A stale connection (e.g. the Space restarted) is the common
            # cause; reconnect on the next request.
            self._reset()
            logger.warning("medgemma request failed", exc_info=exc)
            raise ModelUnavailableError(
                "The AI model is unavailable right now. Please try again shortly."
            ) from exc

        if not reply:
            raise ModelUnavailableError("The AI model returned an empty reply.")
        return reply


medgemma = MedGemmaClient(
    space_id=settings.hf_space_id,
    token=settings.hf_token,
    timeout=settings.model_timeout_seconds,
)
