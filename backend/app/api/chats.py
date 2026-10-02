"""A signed-in patient's chat history: list, open, save, delete."""

from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, Response, status
from sqlmodel import Session

from app.api.deps import current_user
from app.db import get_session
from app.models import User, timestamp
from app.schemas.chat import ChatDetail, ChatImport, ChatSummary
from app.services import chats

router = APIRouter(prefix="/chats", tags=["chat history"])


def _owned(db: Session, user: User, chat_id: str):
    assert user.id is not None
    chat = chats.owned_chat(db, user.id, chat_id)
    if chat is None:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Chat not found.")
    return chat


@router.get("", response_model=list[ChatSummary])
def list_chats(
    user: User = Depends(current_user), db: Session = Depends(get_session)
) -> list[ChatSummary]:
    assert user.id is not None
    return [
        ChatSummary(
            id=chat.id,
            title=chat.title,
            updated_at=timestamp(chat.updated_at),
            message_count=count,
        )
        for chat, count in chats.list_for(db, user.id)
    ]


@router.get("/{chat_id}", response_model=ChatDetail)
def open_chat(
    chat_id: str, user: User = Depends(current_user), db: Session = Depends(get_session)
) -> ChatDetail:
    chat = _owned(db, user, chat_id)
    return ChatDetail(
        id=chat.id,
        title=chat.title,
        updated_at=timestamp(chat.updated_at),
        messages=chats.messages_of(db, chat),
    )


@router.post("", response_model=ChatSummary, status_code=status.HTTP_201_CREATED)
def save_chat(
    body: ChatImport,
    user: User = Depends(current_user),
    db: Session = Depends(get_session),
) -> ChatSummary:
    """Keep a conversation that was started before signing in."""
    assert user.id is not None
    chat = chats.create(db, user.id, body.messages)
    return ChatSummary(
        id=chat.id,
        title=chat.title,
        updated_at=timestamp(chat.updated_at),
        message_count=len(body.messages),
    )


@router.delete(
    "/{chat_id}", status_code=status.HTTP_204_NO_CONTENT, response_class=Response
)
def delete_chat(
    chat_id: str, user: User = Depends(current_user), db: Session = Depends(get_session)
) -> Response:
    chats.delete(db, _owned(db, user, chat_id))
    return Response(status_code=status.HTTP_204_NO_CONTENT)
