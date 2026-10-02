"""Who is making the request: the signed-in user, read from the session cookie.

Every protected endpoint checks this on the server. Hiding a button in the
browser is never the only guard.
"""

from __future__ import annotations

from dataclasses import dataclass

from fastapi import Depends, HTTPException, Request, status
from sqlmodel import Session

from app.db import get_engine, get_session
from app.models import Application, User
from app.schemas.consultation import ProfessionalBadge
from app.services import verification
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


@dataclass
class VerifiedProfessional:
    user: User
    application: Application

    @property
    def badge(self) -> ProfessionalBadge:
        # The name the admin checked against the citizenship certificate.
        return ProfessionalBadge(
            name=self.application.full_name, role=self.application.role
        )


def require_professional(
    user: User = Depends(current_user), db: Session = Depends(get_session)
) -> VerifiedProfessional:
    """A doctor, pharmacist or MBBS student whose application is approved."""
    application = verification.application_for(db, user)
    if application is None or not verification.is_verified(user, application):
        raise HTTPException(
            status.HTTP_403_FORBIDDEN,
            "Only verified doctors, pharmacists and MBBS students can do this.",
        )
    return VerifiedProfessional(user=user, application=application)


def professional_badge_for_session(token: str | None) -> ProfessionalBadge | None:
    """The same check for WebSockets, which run outside FastAPI's
    dependency system here. Blocking: call it through run_in_threadpool."""
    with Session(get_engine()) as db:
        user = user_for_session(db, token)
        if user is None:
            return None
        application = verification.application_for(db, user)
        if application is None or not verification.is_verified(user, application):
            return None
        return VerifiedProfessional(user=user, application=application).badge
