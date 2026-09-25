from __future__ import annotations

from itertools import pairwise

import pytest
from fastapi.testclient import TestClient

from app.ai.medgemma import ModelUnavailableError, medgemma
from app.ai.prompts import SYSTEM_PROMPT, build_model_messages, is_urgent
from app.core.config import settings
from app.schemas.chat import ChatMessage


def say(text: str) -> dict:
    return {"messages": [{"role": "user", "content": text}]}


def test_chat_returns_model_reply(client: TestClient, fake_model: list) -> None:
    response = client.post("/api/v1/chat", json=say("I cut my finger"))

    assert response.status_code == 200
    assert response.json() == {"reply": "1. Stay calm.\n2. Rest.", "urgent": False}
    sent = fake_model[0]
    assert sent[0] == {"role": "system", "content": SYSTEM_PROMPT}
    assert sent[-1] == {"role": "user", "content": "I cut my finger"}


def test_urgent_message_is_flagged(client: TestClient, fake_model: list) -> None:
    body = client.post("/api/v1/chat", json=say("My father has chest pain")).json()
    assert body["urgent"] is True


@pytest.mark.parametrize(
    ("text", "urgent"),
    [
        ("He is unconscious", True),
        ("बुबाको छाती दुख्यो", True),  # father's chest hurts
        ("उहाँ बेहोस हुनुभयो", True),  # he fainted
        ("I have a mild headache", False),
        ("What are the benefits of rest?", False),  # "fits" inside a word
        ("यो विषयमा जानकारी चाहियो", False),  # "topic", contains विष
    ],
)
def test_urgent_keywords(text: str, urgent: bool) -> None:
    assert is_urgent(text) is urgent


def test_model_failure_is_a_clear_503(
    client: TestClient, monkeypatch: pytest.MonkeyPatch
) -> None:
    async def down(messages):
        raise ModelUnavailableError("The AI model is unavailable right now.")

    monkeypatch.setattr(medgemma, "generate", down)

    response = client.post("/api/v1/chat", json=say("hello"))

    assert response.status_code == 503
    assert "unavailable" in response.json()["detail"]


def test_unconfigured_model_is_a_503(client: TestClient) -> None:
    # No HF_SPACE_ID in the test environment, and the real client is used.
    response = client.post("/api/v1/chat", json=say("hello"))
    assert response.status_code == 503
    assert "not configured" in response.json()["detail"]


def test_last_message_must_be_from_user(client: TestClient, fake_model: list) -> None:
    body = {
        "messages": [
            {"role": "user", "content": "hi"},
            {"role": "assistant", "content": "hello"},
        ]
    }
    assert client.post("/api/v1/chat", json=body).status_code == 422


def test_overlong_message_rejected(client: TestClient, fake_model: list) -> None:
    text = "x" * (settings.chat_max_message_chars + 1)
    assert client.post("/api/v1/chat", json=say(text)).status_code == 422
    assert fake_model == []


def test_rate_limit(
    client: TestClient, fake_model: list, monkeypatch: pytest.MonkeyPatch
) -> None:
    from app.services import chat_rate_limiter

    monkeypatch.setattr(chat_rate_limiter, "_limit", 2)
    assert client.post("/api/v1/chat", json=say("a")).status_code == 200
    assert client.post("/api/v1/chat", json=say("b")).status_code == 200
    assert client.post("/api/v1/chat", json=say("c")).status_code == 429


def test_history_is_trimmed_and_alternates() -> None:
    history = [
        ChatMessage(role="user" if i % 2 == 0 else "assistant", content=f"m{i}")
        for i in range(12)
    ] + [ChatMessage(role="user", content="latest")]

    messages = build_model_messages(history, max_messages=4)

    roles = [m["role"] for m in messages]
    assert roles[0] == "system"
    assert roles[1] == "user"  # leading assistant turn dropped after trimming
    assert all(a != b for a, b in pairwise(roles[1:]))
    assert messages[-1]["content"] == "latest"
    assert len(messages) <= 5


def test_consecutive_user_turns_are_merged() -> None:
    history = [
        ChatMessage(role="user", content="first"),
        ChatMessage(role="user", content="second"),
    ]
    messages = build_model_messages(history, max_messages=8)
    assert messages[1] == {"role": "user", "content": "first\n\nsecond"}
