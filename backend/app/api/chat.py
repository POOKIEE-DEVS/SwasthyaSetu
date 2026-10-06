from __future__ import annotations

import logging

from fastapi import APIRouter, Depends, HTTPException, Request, status
from fastapi.concurrency import run_in_threadpool

from app.ai.medgemma import ModelUnavailableError, medgemma
from app.ai.prompts import (
    FALLBACK_SYSTEM_PROMPT,
    build_model_messages,
    is_off_topic,
    is_urgent,
    looks_like_reasoning,
    off_topic_reply,
)
from app.api.deps import optional_user
from app.core.config import settings
from app.models import User
from app.schemas.chat import ChatMessage, ChatRequest, ChatResponse
from app.services import chat_rate_limiter, chats

logger = logging.getLogger(__name__)
router = APIRouter(tags=["chat"])


async def ask_model(history: list[ChatMessage]) -> str:
    """The model's reply, never its analysis.

    If the reply reads as the model working through the question ("The user
    has asked ... Plan: ..."), ask once more with the Space test page's prompt
    and only the latest message. Different input, so a different answer even
    with greedy decoding. If that is analysis too, fail with the friendly
    "couldn't finish an answer" message rather than show it.
    """
    reply = await medgemma.generate(
        build_model_messages(history, max_messages=settings.chat_max_history_messages)
    )
    if not looks_like_reasoning(reply):
        return reply
    logger.warning("model replied with its analysis; retrying with the fallback prompt")
    retry = await medgemma.generate(
        build_model_messages(
            history[-1:], max_messages=1, system_prompt=FALLBACK_SYSTEM_PROMPT
        )
    )
    if looks_like_reasoning(retry):
        raise ModelUnavailableError(
            "The assistant couldn't finish an answer. Please try again, "
            "or press Talk to a professional."
        )
    return retry


@router.post("/chat", response_model=ChatResponse)
async def chat(
    body: ChatRequest,
    request: Request,
    # Optional: guests chat exactly as before. Without a session cookie
    # this never touches the database.
    user: User | None = Depends(optional_user),
) -> ChatResponse:
    client_ip = request.client.host if request.client else "unknown"
    if not chat_rate_limiter.allow(client_ip):
        raise HTTPException(
            status.HTTP_429_TOO_MANY_REQUESTS,
            "Too many messages. Please wait a minute and try again.",
        )

    latest = body.messages[-1]
    if latest.role != "user":
        raise HTTPException(
            status.HTTP_422_UNPROCESSABLE_CONTENT,
            "The last message must be from the user.",
        )
    if any(len(m.content) > settings.chat_max_message_chars for m in body.messages):
        raise HTTPException(
            status.HTTP_422_UNPROCESSABLE_CONTENT,
            f"Messages must be under {settings.chat_max_message_chars} characters.",
        )

    urgent = is_urgent(latest.content)
    if is_off_topic(latest.content):
        # Plainly not about health: a fixed answer, and the model never sees
        # it (see is_off_topic; anything health-related or urgent goes on).
        reply = off_topic_reply(latest.content)
    else:
        try:
            reply = await ask_model(body.messages)
        except ModelUnavailableError as exc:
            raise HTTPException(
                status.HTTP_503_SERVICE_UNAVAILABLE, str(exc)
            ) from exc

    chat_id = None
    if user is not None and user.id is not None:
        try:
            chat_id = await run_in_threadpool(
                chats.save_exchange, user.id, body.chat_id, body.messages, reply
            )
        except Exception as exc:
            # The reply matters more than the history: never fail the
            # answer because saving it did.
            logger.warning("could not save chat history", exc_info=exc)

    return ChatResponse(reply=reply, urgent=urgent, chat_id=chat_id)
