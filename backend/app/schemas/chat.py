from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field


class ChatMessage(BaseModel):
    role: Literal["user", "assistant"]
    content: str = Field(min_length=1)


class ChatRequest(BaseModel):
    # The client holds the conversation and sends it with every request, so
    # the model call stays stateless and guests need no account.
    messages: list[ChatMessage] = Field(min_length=1, max_length=100)
    # Signed-in patients only: the saved chat this message continues.
    chat_id: str | None = Field(default=None, max_length=32)


class ChatResponse(BaseModel):
    reply: str
    # True when the user's latest message mentions an obvious emergency, so
    # the UI can show "call 102 / talk to a doctor" without waiting on it.
    urgent: bool
    # Set when the exchange was saved to the signed-in patient's history.
    chat_id: str | None = None


class ChatSummary(BaseModel):
    id: str
    title: str
    updated_at: float
    message_count: int


class ChatDetail(BaseModel):
    id: str
    title: str
    updated_at: float
    messages: list[ChatMessage]


class ChatImport(BaseModel):
    """A conversation started as a guest, saved after signing in."""

    messages: list[ChatMessage] = Field(min_length=1, max_length=100)
