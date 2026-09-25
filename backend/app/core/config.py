"""Runtime configuration, loaded from environment variables.

Everything that differs between environments — connection strings, model
endpoints, secrets — enters the process through the environment and nowhere
else. See ``.env.example`` at the service root for the documented set.

Import :data:`settings` for normal use; construct :class:`Settings` directly
in tests that need overrides.
"""

from __future__ import annotations

from functools import cached_property
from typing import Literal

from pydantic import Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


def _split_csv(value: object) -> object:
    """Allow list-valued settings to be given as ``a,b,c`` in a .env file.

    pydantic-settings parses complex types as JSON by default, which is
    hostile to hand-edited env files. Plain comma-separated strings are what
    people actually write, so accept those too.
    """
    if isinstance(value, str) and not value.strip().startswith("["):
        return [item.strip() for item in value.split(",") if item.strip()]
    return value


class Settings(BaseSettings):
    """Validated runtime settings for the backend process.

    Defaults are safe for local development against the docker-compose stack.
    Anything secret defaults to an obvious placeholder so a misconfigured
    deployment fails visibly rather than silently running insecure.
    """

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
        case_sensitive=False,
    )

    # --- Application ---------------------------------------------------
    service_name: str = "backend"
    environment: Literal["development", "staging", "production"] = "development"
    debug: bool = False
    api_v1_prefix: str = "/api/v1"
    host: str = "0.0.0.0"
    port: int = 8000
    log_level: str = "INFO"
    log_json: bool = False
    cors_origins: list[str] = Field(default_factory=lambda: ["http://localhost:3000"])

    # --- PostgreSQL ----------------------------------------------------
    # Async driver (asyncpg) for the request path; a sync URL is derived
    # from this for Alembic and Celery via `sync_database_url`.
    database_url: str = "postgresql+asyncpg://swasthyasetu:dev-only-password@localhost:5432/swasthyasetu"
    db_pool_size: int = 10
    db_max_overflow: int = 5
    db_pool_recycle_seconds: int = 1800
    db_echo: bool = False

    # --- Redis (cache, sessions, Celery broker) ------------------------
    redis_url: str = "redis://localhost:6379/0"
    redis_max_connections: int = 20

    # --- Celery --------------------------------------------------------
    # Separate logical databases so a cache flush never drops queued work.
    celery_broker_url: str = "redis://localhost:6379/1"
    celery_result_backend: str = "redis://localhost:6379/2"
    celery_task_soft_time_limit: int = 120
    celery_task_time_limit: int = 180

    # --- Security (consumed from Week 3, when auth lands) --------------
    jwt_secret: str = "changeme-in-local-env"
    jwt_algorithm: str = "HS256"
    access_token_ttl_minutes: int = 15
    refresh_token_ttl_days: int = 30
    password_hash_scheme: str = "argon2"

    # --- AI & Edge ML engine (consumed from Week 9) --------------------
    # Clinical triage model. Served locally through Ollama in development;
    # point `medgemma_base_url` at a hosted endpoint in production.
    medgemma_model: str = "medgemma"
    medgemma_base_url: str = "http://localhost:11434"
    medgemma_api_key: str = ""
    medgemma_timeout_seconds: float = 60.0
    medgemma_max_tokens: int = 512
    # Deterministic by default: medical guidance should not vary run to run.
    medgemma_temperature: float = 0.0

    # Speech-to-text — whisper.cpp, run on-device for Nepali voice input.
    whisper_cpp_binary: str = "/usr/local/bin/whisper-cli"
    whisper_model_path: str = "/models/ggml-base-q5_1.bin"
    whisper_language: str = "ne"
    whisper_threads: int = 4

    # Text-to-speech — localized Nepali narration of first-aid guidance.
    tts_engine: str = "piper"
    tts_model_path: str = "/models/ne_NP-voice.onnx"
    tts_output_format: str = "wav"

    # --- WebRTC teleconsultation ---------------------------------------
    stun_urls: list[str] = Field(
        default_factory=lambda: ["stun:stun.l.google.com:19302"]
    )
    turn_urls: list[str] = Field(default_factory=lambda: ["turn:localhost:3478"])
    turn_username: str = "swasthyasetu"
    turn_credential: str = "changeme-in-local-env"
    # How long an issued ICE credential stays valid for a joining client.
    turn_credential_ttl_seconds: int = 3600

    _split = field_validator("cors_origins", "stun_urls", "turn_urls", mode="before")(
        _split_csv
    )

    @cached_property
    def sync_database_url(self) -> str:
        """The same database, over a blocking driver.

        Alembic and Celery workers run outside the async request path; rather
        than maintain a second URL that can drift out of sync, swap the driver
        on the one that is configured.
        """
        return self.database_url.replace("+asyncpg", "+psycopg")

    @cached_property
    def is_production(self) -> bool:
        return self.environment == "production"


settings = Settings()
