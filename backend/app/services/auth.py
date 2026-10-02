"""Sign-in: Google OAuth (authorization code + PKCE) and login sessions.

Flow:
1. ``start_google_login`` makes a random ``state`` and PKCE verifier, keeps
   them in memory for 10 minutes, and returns Google's consent-screen URL.
   The state also goes into a short-lived cookie, which ties the callback
   to the browser that started it (stops login CSRF).
2. Google redirects back with ``code`` and ``state``. ``finish_google_login``
   checks the state, exchanges the code (with the client secret and PKCE
   verifier) for an ID token, and reads the profile from it.
3. ``upsert_google_user`` creates or updates the user, and
   ``create_session`` issues the session cookie.

The ID token comes straight from Google's token endpoint over TLS,
authenticated with our client secret, so Google's guidance is that its
signature need not be re-verified; its issuer, audience, expiry and
verified-email claims are still checked.
"""

from __future__ import annotations

import base64
import hashlib
import json
import secrets
import time
from dataclasses import dataclass, field
from datetime import timedelta
from urllib.parse import urlencode

import httpx
from sqlmodel import Session, select

from app.core.config import settings
from app.models import LoginSession, User, utcnow

SESSION_COOKIE = "swasthya_session"
STATE_COOKIE = "swasthya_oauth_state"

GOOGLE_AUTH_URL = "https://accounts.google.com/o/oauth2/v2/auth"
GOOGLE_TOKEN_URL = "https://oauth2.googleapis.com/token"
GOOGLE_ISSUERS = ("accounts.google.com", "https://accounts.google.com")

_LOGIN_TTL_SECONDS = 600
_MAX_PENDING_LOGINS = 1000


class OAuthError(Exception):
    """Sign-in failed. The message is safe to show to the user."""


# --- Sessions ---------------------------------------------------------------


def _hash(token: str) -> str:
    return hashlib.sha256(token.encode()).hexdigest()


def create_session(db: Session, user: User) -> str:
    token = secrets.token_urlsafe(32)
    db.add(
        LoginSession(
            token_hash=_hash(token),
            user_id=user.id,
            expires_at=utcnow() + timedelta(days=settings.session_days),
        )
    )
    db.commit()
    return token


def user_for_session(db: Session, token: str | None) -> User | None:
    if not token:
        return None
    row = db.get(LoginSession, _hash(token))
    if row is None or row.expires_at < utcnow():
        return None
    return db.get(User, row.user_id)


def delete_session(db: Session, token: str | None) -> None:
    if not token:
        return
    row = db.get(LoginSession, _hash(token))
    if row is not None:
        db.delete(row)
        db.commit()


def is_admin(user: User) -> bool:
    return user.email.lower() in settings.admin_email_list


def safe_next(value: str | None) -> str:
    """Only same-site paths, so the login can't redirect to another site."""
    if (
        value
        and value.startswith("/")
        and not value.startswith("//")
        and "\\" not in value
    ):
        return value
    return "/"


# --- Google OAuth -----------------------------------------------------------


@dataclass
class _PendingLogin:
    verifier: str
    next_path: str
    created: float = field(default_factory=time.monotonic)


# In memory: the backend runs as one process, and a login that spans a
# restart just has to be retried.
_pending: dict[str, _PendingLogin] = {}


def _purge_pending() -> None:
    cutoff = time.monotonic() - _LOGIN_TTL_SECONDS
    for state in [s for s, p in _pending.items() if p.created < cutoff]:
        del _pending[state]
    while len(_pending) > _MAX_PENDING_LOGINS:
        del _pending[next(iter(_pending))]


def start_google_login(redirect_uri: str, next_path: str) -> tuple[str, str]:
    """Returns (Google consent URL, state)."""
    _purge_pending()
    state = secrets.token_urlsafe(24)
    verifier = secrets.token_urlsafe(48)
    challenge = (
        base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest())
        .rstrip(b"=")
        .decode()
    )
    _pending[state] = _PendingLogin(verifier=verifier, next_path=safe_next(next_path))
    query = urlencode(
        {
            "client_id": settings.google_client_id,
            "redirect_uri": redirect_uri,
            "response_type": "code",
            "scope": "openid email profile",
            "state": state,
            "code_challenge": challenge,
            "code_challenge_method": "S256",
            "prompt": "select_account",
        }
    )
    return f"{GOOGLE_AUTH_URL}?{query}", state


def take_pending_login(state: str, cookie_state: str | None) -> _PendingLogin:
    if not cookie_state or not secrets.compare_digest(state, cookie_state):
        raise OAuthError("Sign-in expired or was started in another tab. Try again.")
    _purge_pending()
    pending = _pending.pop(state, None)
    if pending is None:
        raise OAuthError("Sign-in expired. Please try again.")
    return pending


async def exchange_code(code: str, verifier: str, redirect_uri: str) -> dict:
    """Swap the authorization code for tokens at Google's token endpoint."""
    async with httpx.AsyncClient(timeout=10) as client:
        response = await client.post(
            GOOGLE_TOKEN_URL,
            data={
                "code": code,
                "client_id": settings.google_client_id,
                "client_secret": settings.google_client_secret,
                "redirect_uri": redirect_uri,
                "grant_type": "authorization_code",
                "code_verifier": verifier,
            },
        )
    if response.status_code != 200:
        raise OAuthError("Google sign-in failed. Please try again.")
    return response.json()


@dataclass
class GoogleProfile:
    sub: str
    email: str
    name: str
    picture: str | None


def profile_from_id_token(id_token: str) -> GoogleProfile:
    try:
        payload_b64 = id_token.split(".")[1]
        payload_b64 += "=" * (-len(payload_b64) % 4)
        claims = json.loads(base64.urlsafe_b64decode(payload_b64))
    except (IndexError, ValueError) as exc:
        raise OAuthError("Google sign-in failed. Please try again.") from exc

    if claims.get("iss") not in GOOGLE_ISSUERS:
        raise OAuthError("Google sign-in failed (issuer).")
    if claims.get("aud") != settings.google_client_id:
        raise OAuthError("Google sign-in failed (audience).")
    if float(claims.get("exp", 0)) < time.time():
        raise OAuthError("Google sign-in expired. Please try again.")
    if not claims.get("sub") or not claims.get("email"):
        raise OAuthError("Google did not share an email address.")
    if claims.get("email_verified") is not True:
        raise OAuthError("Please use a Google account with a verified email.")

    return GoogleProfile(
        sub=str(claims["sub"]),
        email=str(claims["email"]).lower(),
        name=str(claims.get("name") or claims["email"]).strip()[:120],
        picture=claims.get("picture"),
    )


def upsert_google_user(db: Session, profile: GoogleProfile) -> User:
    user = db.exec(select(User).where(User.google_sub == profile.sub)).first()
    if user is None:
        # Same email signed in before without Google (development login).
        user = db.exec(select(User).where(User.email == profile.email)).first()
    if user is None:
        user = User(email=profile.email, name=profile.name)
    user.google_sub = profile.sub
    user.email = profile.email
    user.name = profile.name
    user.picture_url = profile.picture
    db.add(user)
    db.commit()
    db.refresh(user)
    return user


def upsert_dev_user(db: Session, email: str, name: str) -> User:
    email = email.strip().lower()
    user = db.exec(select(User).where(User.email == email)).first()
    if user is None:
        user = User(email=email, name=name.strip())
        db.add(user)
        db.commit()
        db.refresh(user)
    return user


def reset_pending_logins() -> None:
    """For tests."""
    _pending.clear()
