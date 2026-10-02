"""Runtime configuration, loaded from environment variables (or backend/.env).

List-valued settings are plain comma-separated strings with parsing
properties, not ``list[str]`` fields. pydantic-settings JSON-decodes complex
field types *before* any validator runs, so ``CORS_ORIGINS=http://a,http://b``
would crash startup. Strings avoid that failure mode entirely.
"""

from __future__ import annotations

from pathlib import Path
from typing import Literal

from pydantic import AliasChoices, Field
from pydantic_settings import BaseSettings, SettingsConfigDict


def split_csv(value: str) -> list[str]:
    return [item.strip() for item in value.split(",") if item.strip()]


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
        case_sensitive=False,
    )

    # --- Application -----------------------------------------------------
    environment: Literal["development", "production"] = "development"
    log_level: str = "INFO"
    log_json: bool = False
    # Only needed when the frontend runs on a different origin (next dev on
    # :3000). In production the frontend is served by this app, same origin.
    cors_origins: str = "http://localhost:3000"
    # Directory holding the Next.js static export. Served at "/" when present.
    static_dir: str = "static"

    # --- AI model (MedGemma on a Hugging Face Space) ---------------------
    # A Space ID ("your-username/swasthyasetu-medgemma") or the URL of any
    # running copy of model-space/app.py, e.g. a Colab "https://….gradio.live".
    hf_space_id: str = ""
    hf_token: str = ""  # needed for a private Space; leave empty for a URL
    model_timeout_seconds: float = 120.0
    # The model card notes MedGemma is not optimised for long multi-turn
    # conversations, so only recent turns are sent.
    chat_max_history_messages: int = 8
    chat_max_message_chars: int = 2000
    chat_rate_limit_per_minute: int = 15

    # --- WebRTC ICE servers ------------------------------------------------
    stun_urls: str = "stun:stun.l.google.com:19302,stun:stun.cloudflare.com:3478"
    # Option A (recommended): Cloudflare TURN mints short-lived credentials.
    cloudflare_turn_key_id: str = ""
    cloudflare_turn_api_token: str = ""
    turn_credential_ttl_seconds: int = 86400
    # Option B: any TURN provider with static credentials (e.g. Metered).
    turn_urls: str = ""
    turn_username: str = ""
    turn_credential: str = ""

    # --- Consultations ---------------------------------------------------
    # Unanswered or finished requests are dropped from memory after this.
    consultation_ttl_minutes: int = 60

    # --- Database ----------------------------------------------------------
    # Postgres in production (a Neon or Supabase connection string). The
    # SQLite default is for local development only: Render's disk is wiped
    # on every restart, so accounts and approvals stored there would vanish.
    database_url: str = "sqlite:///./swasthyasetu.db"

    # --- Sign-in (Google OAuth) ----------------------------------------
    google_client_id: str = ""
    google_client_secret: str = ""
    # The site's public base URL, e.g. https://swasthyasetu-580o.onrender.com.
    # Google redirects back to {PUBLIC_URL}/api/v1/auth/google/callback.
    # Render sets RENDER_EXTERNAL_URL itself, so on Render this needs no
    # setting. Empty: derived from the request.
    public_url: str = Field(
        default="", validation_alias=AliasChoices("PUBLIC_URL", "RENDER_EXTERNAL_URL")
    )
    # Comma-separated Google account emails allowed into the admin review page.
    admin_emails: str = ""
    session_days: int = 30
    # Local development and the smoke test only: sign in without Google.
    # Always off when ENVIRONMENT=production.
    dev_login: bool = False

    # --- Professional verification documents ---------------------------
    max_upload_bytes: int = 5 * 1024 * 1024

    @property
    def cors_origin_list(self) -> list[str]:
        return split_csv(self.cors_origins)

    @property
    def stun_url_list(self) -> list[str]:
        return split_csv(self.stun_urls)

    @property
    def turn_url_list(self) -> list[str]:
        return split_csv(self.turn_urls)

    @property
    def static_path(self) -> Path:
        return Path(self.static_dir)

    @property
    def model_configured(self) -> bool:
        return bool(self.hf_space_id)

    @property
    def is_production(self) -> bool:
        return self.environment == "production"

    @property
    def admin_email_list(self) -> list[str]:
        return [email.lower() for email in split_csv(self.admin_emails)]

    @property
    def google_configured(self) -> bool:
        return bool(self.google_client_id and self.google_client_secret)

    @property
    def dev_login_enabled(self) -> bool:
        return self.dev_login and not self.is_production

    @property
    def sqlalchemy_database_url(self) -> str:
        """Neon and Supabase hand out ``postgres://`` / ``postgresql://`` URLs;
        SQLAlchemy needs the driver named to use psycopg 3."""
        url = self.database_url.strip()
        for prefix in ("postgres://", "postgresql://"):
            if url.startswith(prefix):
                return "postgresql+psycopg://" + url[len(prefix) :]
        return url

    @property
    def database_kind(self) -> Literal["postgres", "sqlite"]:
        if self.sqlalchemy_database_url.startswith("sqlite"):
            return "sqlite"
        return "postgres"


settings = Settings()
