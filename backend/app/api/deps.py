"""Who is making the request: the signed-in user, read from the session cookie.

Every protected endpoint checks this on the server. Hiding a button in the
browser is never the only guard.
"""

from __future__ import annotations

from fastapi import Depends, HTTPException, Request, status
from sqlmodel import Session

from app.db import get_session
from app.models import User
from app.services.auth import SESSION_COOKIE, is_admin, user_for_session


def optional_user(request: Request, db: Session = Depends(get_session)) -> User | None:
    return user_for_session(db, request.cookies.get(SESSION_COOKIE))


def current_user(user: User | None = Depends(optional_user)) -> User:
    if user is None:
        raise HTTPException(status.HTTP_401_UNAUTHORIZED, "Please sign in first.")
    return user


def require_admin(user: User = Depends(current_user)) -> User:
    if not is_admin(user):
        raise HTTPException(status.HTTP_403_FORBIDDEN, "Admins only.")
    return user
