from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field


class ChatMessage(BaseModel):
    role: Literal["user", "assistant"]
    content: str = Field(min_length=1)


class ChatRequest(BaseModel):
    # The client holds the conversation and sends it with every request, so
    # the server stays stateless and a restart loses nothing it owned.
    messages: list[ChatMessage] = Field(min_length=1, max_length=100)


class ChatResponse(BaseModel):
    reply: str
    # True when the user's latest message mentions an obvious emergency, so
    # the UI can show "call 102 / talk to a doctor" without waiting on it.
    urgent: bool
