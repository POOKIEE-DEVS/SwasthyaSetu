from __future__ import annotations

import pytest
from fastapi.testclient import TestClient

from tests.conftest import sign_in


def ask(client: TestClient, messages: list[dict], chat_id: str | None = None) -> dict:
    response = client.post(
        "/api/v1/chat", json={"messages": messages, "chat_id": chat_id}
    )
    assert response.status_code == 200, response.text
    return response.json()


def user(text: str) -> dict:
    return {"role": "user", "content": text}


def test_guest_chat_is_not_saved(client: TestClient, fake_model: list) -> None:
    body = ask(client, [user("I burned my hand")])
    assert body["chat_id"] is None
    assert client.get("/api/v1/chats").status_code == 401


def test_signed_in_chat_is_saved_and_continued(
    client: TestClient, fake_model: list
) -> None:
    sign_in(client, "sita@example.com")
    first = ask(client, [user("I burned my hand while cooking")])
    chat_id = first["chat_id"]
    assert chat_id

    history = [
        user("I burned my hand while cooking"),
        {"role": "assistant", "content": first["reply"]},
        user("Should I put ice on it?"),
    ]
    second = ask(client, history, chat_id)
    assert second["chat_id"] == chat_id

    listed = client.get("/api/v1/chats").json()
    assert len(listed) == 1
    assert listed[0]["title"] == "I burned my hand while cooking"
    assert listed[0]["message_count"] == 4

    opened = client.get(f"/api/v1/chats/{chat_id}").json()
    assert [m["role"] for m in opened["messages"]] == [
        "user",
        "assistant",
        "user",
        "assistant",
    ]
    assert opened["messages"][2]["content"] == "Should I put ice on it?"


def test_failed_reply_saves_nothing_and_retry_does_not_duplicate(
    client: TestClient, fake_model: list, monkeypatch: pytest.MonkeyPatch
) -> None:
    from app.ai.medgemma import ModelUnavailableError, medgemma

    sign_in(client, "sita@example.com")
    chat_id = ask(client, [user("Fever")])["chat_id"]
    history = [
        user("Fever"),
        {"role": "assistant", "content": "1. Rest."},
        user("Still hot"),
    ]

    async def down(_messages: list) -> str:
        raise ModelUnavailableError("down")

    working = medgemma.generate
    monkeypatch.setattr(medgemma, "generate", down)
    failed = client.post("/api/v1/chat", json={"messages": history, "chat_id": chat_id})
    assert failed.status_code == 503
    monkeypatch.setattr(medgemma, "generate", working)

    ask(client, history, chat_id)  # the retry
    contents = [
        m["content"] for m in client.get(f"/api/v1/chats/{chat_id}").json()["messages"]
    ]
    assert contents.count("Still hot") == 1
    assert len(contents) == 4


def test_new_chat_starts_a_new_entry(client: TestClient, fake_model: list) -> None:
    sign_in(client, "sita@example.com")
    a = ask(client, [user("Headache")])["chat_id"]
    b = ask(client, [user("Cough")])["chat_id"]
    assert a != b
    titles = [c["title"] for c in client.get("/api/v1/chats").json()]
    assert titles == ["Cough", "Headache"]  # newest first


def test_guest_chat_can_be_saved_after_sign_in(client: TestClient) -> None:
    sign_in(client, "sita@example.com")
    saved = client.post(
        "/api/v1/chats",
        json={
            "messages": [
                user("Snake bite on leg"),
                {"role": "assistant", "content": "Call 102."},
            ]
        },
    )
    assert saved.status_code == 201
    assert saved.json()["message_count"] == 2
    assert client.get(f"/api/v1/chats/{saved.json()['id']}").status_code == 200


def test_chats_are_private(client: TestClient, fake_model: list) -> None:
    sign_in(client, "sita@example.com")
    chat_id = ask(client, [user("Private question")])["chat_id"]

    client.cookies.clear()
    sign_in(client, "ram@example.com")
    assert client.get("/api/v1/chats").json() == []
    assert client.get(f"/api/v1/chats/{chat_id}").status_code == 404
    assert client.delete(f"/api/v1/chats/{chat_id}").status_code == 404
    # Continuing someone else's chat id starts a fresh chat of your own.
    mine = ask(client, [user("My question")], chat_id)["chat_id"]
    assert mine != chat_id

    client.cookies.clear()
    sign_in(client, "sita@example.com")
    messages = client.get(f"/api/v1/chats/{chat_id}").json()["messages"]
    assert messages[0]["content"] == "Private question"
    assert len(messages) == 2


def test_delete_chat(client: TestClient, fake_model: list) -> None:
    sign_in(client, "sita@example.com")
    chat_id = ask(client, [user("Headache")])["chat_id"]
    assert client.delete(f"/api/v1/chats/{chat_id}").status_code == 204
    assert client.get("/api/v1/chats").json() == []


def test_long_first_message_makes_a_short_title(
    client: TestClient, fake_model: list
) -> None:
    sign_in(client, "sita@example.com")
    ask(client, [user("word " * 50)])
    title = client.get("/api/v1/chats").json()[0]["title"]
    assert len(title) == 60
    assert title.endswith("…")


def test_reply_survives_a_history_failure(
    client: TestClient, fake_model: list, monkeypatch: pytest.MonkeyPatch
) -> None:
    from app.services import chats

    def broken(*_args: object) -> str:
        raise RuntimeError("database down")

    monkeypatch.setattr(chats, "save_exchange", broken)
    sign_in(client, "sita@example.com")
    body = ask(client, [user("Burn")])
    assert body["reply"]
    assert body["chat_id"] is None
