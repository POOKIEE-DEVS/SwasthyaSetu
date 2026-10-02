"""Sign in with Google, sign out, who am I, and choosing a role.

Signing in is optional for patients: chat and "talk to a doctor" work
without it. It is required for medical professionals (to apply and, once
verified, to take calls) and for the admin.
"""

from __future__ import annotations

from urllib.parse import urlencode

from fastapi import APIRouter, Depends, HTTPException, Request, Response, status
from fastapi.concurrency import run_in_threadpool
from fastapi.responses import RedirectResponse
from sqlmodel import Session

from app.api.deps import current_user, optional_user
from app.core.config import settings
from app.db import get_engine, get_session
from app.models import User
from app.schemas.auth import (
    DevLoginRequest,
    MeResponse,
    RoleChoice,
    UserPublic,
)
from app.services.auth import (
    SESSION_COOKIE,
    STATE_COOKIE,
    GoogleProfile,
    OAuthError,
    create_session,
    delete_session,
    exchange_code,
    is_admin,
    profile_from_id_token,
    start_google_login,
    take_pending_login,
    upsert_dev_user,
    upsert_google_user,
)

router = APIRouter(prefix="/auth", tags=["auth"])

STATE_COOKIE_PATH = "/api/v1/auth"


def user_public(user: User) -> UserPublic:
    assert user.id is not None
    return UserPublic(
        id=user.id,
        email=user.email,
        name=user.name,
        picture_url=user.picture_url,
        role=user.role,
        is_admin=is_admin(user),
    )


def _redirect_uri(request: Request) -> str:
    # Behind Render's proxy, uvicorn's --proxy-headers makes the request
    # scheme https, so the derived URL matches what Google has registered.
    base = settings.public_url.strip().rstrip("/") or str(request.base_url).rstrip("/")
    return f"{base}/api/v1/auth/google/callback"


def _set_session_cookie(response: Response, request: Request, token: str) -> None:
    response.set_cookie(
        SESSION_COOKIE,
        token,
        max_age=settings.session_days * 24 * 3600,
        httponly=True,
        secure=request.url.scheme == "https",
        # Lax: sent on normal navigation, never on cross-site form posts.
        samesite="lax",
        path="/",
    )


@router.get("/me", response_model=MeResponse)
def me(user: User | None = Depends(optional_user)) -> MeResponse:
    return MeResponse(
        user=user_public(user) if user else None,
        google_enabled=settings.google_configured,
        dev_login=settings.dev_login_enabled,
    )


@router.get("/google/login")
def google_login(request: Request, next: str = "/") -> RedirectResponse:
    if not settings.google_configured:
        raise HTTPException(
            status.HTTP_404_NOT_FOUND, "Google sign-in is not configured."
        )
    url, state = start_google_login(_redirect_uri(request), next)
    response = RedirectResponse(url, status_code=status.HTTP_303_SEE_OTHER)
    response.set_cookie(
        STATE_COOKIE,
        state,
        max_age=600,
        httponly=True,
        secure=request.url.scheme == "https",
        samesite="lax",
        path=STATE_COOKIE_PATH,
    )
    return response


def _sign_in(profile: GoogleProfile) -> tuple[str, bool]:
    """Returns (session token, whether the user still has to pick a role)."""
    with Session(get_engine()) as db:
        user = upsert_google_user(db, profile)
        return create_session(db, user), user.role is None


@router.get("/google/callback")
async def google_callback(
    request: Request, code: str = "", state: str = "", error: str = ""
) -> RedirectResponse:
    try:
        pending = take_pending_login(state, request.cookies.get(STATE_COOKIE))
        if error or not code:
            raise OAuthError("Sign-in was cancelled.")
        tokens = await exchange_code(code, pending.verifier, _redirect_uri(request))
        profile = profile_from_id_token(str(tokens.get("id_token", "")))
    except OAuthError as exc:
        response = RedirectResponse(
            "/account/?" + urlencode({"error": str(exc)}),
            status_code=status.HTTP_303_SEE_OTHER,
        )
        response.delete_cookie(STATE_COOKIE, path=STATE_COOKIE_PATH)
        return response

    token, needs_role = await run_in_threadpool(_sign_in, profile)
    destination = (
        "/account/?" + urlencode({"next": pending.next_path})
        if needs_role
        else pending.next_path
    )
    response = RedirectResponse(destination, status_code=status.HTTP_303_SEE_OTHER)
    _set_session_cookie(response, request, token)
    response.delete_cookie(STATE_COOKIE, path=STATE_COOKIE_PATH)
    return response


@router.post("/logout", status_code=status.HTTP_204_NO_CONTENT)
def logout(request: Request, db: Session = Depends(get_session)) -> Response:
    delete_session(db, request.cookies.get(SESSION_COOKIE))
    response = Response(status_code=status.HTTP_204_NO_CONTENT)
    response.delete_cookie(SESSION_COOKIE, path="/")
    return response


@router.post("/role", response_model=UserPublic)
def choose_role(
    body: RoleChoice,
    user: User = Depends(current_user),
    db: Session = Depends(get_session),
) -> UserPublic:
    user.role = body.role
    db.add(user)
    db.commit()
    db.refresh(user)
    return user_public(user)


@router.post("/dev-login", response_model=UserPublic)
def dev_login(
    body: DevLoginRequest,
    request: Request,
    response: Response,
    db: Session = Depends(get_session),
) -> UserPublic:
    """Sign in without Google, for local development and the smoke test.
    Disabled unless DEV_LOGIN=true, and always disabled in production."""
    if not settings.dev_login_enabled:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Not found.")
    user = upsert_dev_user(db, body.email, body.name)
    _set_session_cookie(response, request, create_session(db, user))
    return user_public(user)
