"""Saved conversations for signed-in patients.

Guests chat exactly as before: nothing is stored, and the conversation
lives in their browser tab. For a signed-in patient, each successful
reply is saved together with the messages that led to it. A failed model
call saves nothing, so a retry never duplicates a message.

Every query filters by the owner's user id. A chat id belonging to someone
else is treated as unknown, never as forbidden, so ids can't be probed.
"""

from __future__ import annotations

import secrets

from sqlalchemy import delete as sql_delete
from sqlmodel import Session, col, func, select

from app.db import get_engine
from app.models import Chat, ChatEntry, utcnow
from app.schemas.chat import ChatMessage

MAX_LISTED = 50
TITLE_CHARS = 60


def title_for(messages: list[ChatMessage]) -> str:
    first = next((m.content for m in messages if m.role == "user"), "New chat")
    text = " ".join(first.split())
    return text if len(text) <= TITLE_CHARS else text[: TITLE_CHARS - 1] + "…"


def owned_chat(db: Session, user_id: int, chat_id: str | None) -> Chat | None:
    if not chat_id:
        return None
    chat = db.get(Chat, chat_id)
    return chat if chat is not None and chat.user_id == user_id else None


def _entry_count(db: Session, chat_id: str) -> int:
    return db.exec(
        select(func.count()).select_from(ChatEntry).where(ChatEntry.chat_id == chat_id)
    ).one()


def _add_entries(db: Session, chat: Chat, messages: list[ChatMessage]) -> None:
    for message in messages:
        db.add(ChatEntry(chat_id=chat.id, role=message.role, content=message.content))
    chat.updated_at = utcnow()
    db.add(chat)


def create(db: Session, user_id: int, messages: list[ChatMessage]) -> Chat:
    chat = Chat(id=secrets.token_hex(16), user_id=user_id, title=title_for(messages))
    db.add(chat)
    db.flush()
    _add_entries(db, chat, messages)
    db.commit()
    db.refresh(chat)
    return chat


def save_exchange(
    user_id: int, chat_id: str | None, history: list[ChatMessage], reply: str
) -> str:
    """Store the conversation up to and including the new reply; returns the
    chat id. Blocking: call it through run_in_threadpool."""
    with Session(get_engine()) as db:
        conversation = [*history, ChatMessage(role="assistant", content=reply)]
        chat = owned_chat(db, user_id, chat_id)
        if chat is None:
            return create(db, user_id, conversation).id
        # Add only what isn't stored yet (normally the latest question and
        # the reply; more if an earlier question had failed).
        stored = _entry_count(db, chat.id)
        new = conversation[stored:] if stored < len(conversation) else conversation[-2:]
        _add_entries(db, chat, new)
        db.commit()
        return chat.id


def list_for(db: Session, user_id: int) -> list[tuple[Chat, int]]:
    count = (
        select(func.count())
        .select_from(ChatEntry)
        .where(ChatEntry.chat_id == Chat.id)
        .scalar_subquery()
    )
    # Tie-break on the newest message's id (always increasing): two chats
    # can share a timestamp when the clock is coarse (about 15 ms on Windows).
    latest_entry = (
        select(func.max(ChatEntry.id))
        .where(ChatEntry.chat_id == Chat.id)
        .scalar_subquery()
    )
    rows = db.exec(
        select(Chat, count)
        .where(Chat.user_id == user_id)
        .order_by(col(Chat.updated_at).desc(), latest_entry.desc())
        .limit(MAX_LISTED)
    ).all()
    return [(chat, n) for chat, n in rows]


def messages_of(db: Session, chat: Chat) -> list[ChatMessage]:
    entries = db.exec(
        select(ChatEntry)
        .where(ChatEntry.chat_id == chat.id)
        .order_by(col(ChatEntry.id))
    ).all()
    return [ChatMessage(role=e.role, content=e.content) for e in entries]


def delete(db: Session, chat: Chat) -> None:
    # Messages first, in one statement: there is no ORM relationship to
    # tell SQLAlchemy the order, and the parent must go last.
    db.exec(sql_delete(ChatEntry).where(col(ChatEntry.chat_id) == chat.id))
    db.delete(chat)
    db.commit()
