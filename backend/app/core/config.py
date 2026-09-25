"""Runtime configuration, loaded from environment variables (or backend/.env).

List-valued settings are plain comma-separated strings with parsing
properties, not ``list[str]`` fields. pydantic-settings JSON-decodes complex
field types *before* any validator runs, so ``CORS_ORIGINS=http://a,http://b``
would crash startup. Strings avoid that failure mode entirely.
"""

from __future__ import annotations

from pathlib import Path
from typing import Literal

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
    hf_space_id: str = ""  # e.g. "your-username/swasthyasetu-medgemma"
    hf_token: str = ""  # needed when the Space is private (recommended)
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


settings = Settings()
