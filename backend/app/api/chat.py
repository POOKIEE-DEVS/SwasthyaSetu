from __future__ import annotations

import logging

from fastapi import APIRouter, HTTPException, Request, status

from app.ai.medgemma import ModelUnavailableError, medgemma
from app.ai.prompts import build_model_messages, is_urgent
from app.core.config import settings
from app.schemas.chat import ChatRequest, ChatResponse
from app.services import chat_rate_limiter

logger = logging.getLogger(__name__)
router = APIRouter(tags=["chat"])


@router.post("/chat", response_model=ChatResponse)
async def chat(body: ChatRequest, request: Request) -> ChatResponse:
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
    model_messages = build_model_messages(
        body.messages, max_messages=settings.chat_max_history_messages
    )

    try:
        reply = await medgemma.generate(model_messages)
    except ModelUnavailableError as exc:
        raise HTTPException(status.HTTP_503_SERVICE_UNAVAILABLE, str(exc)) from exc

    return ChatResponse(reply=reply, urgent=urgent)
