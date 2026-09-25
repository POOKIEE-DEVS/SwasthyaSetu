# Build stage -- compile wheels once so the runtime image needs no toolchain.
FROM python:3.12-slim AS build

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

RUN apt-get update \
    && apt-get install --no-install-recommends -y build-essential \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip wheel --no-cache-dir --wheel-dir /wheels -r requirements.txt

# Runtime stage.
FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PORT=8000 \
    WEB_CONCURRENCY=4

WORKDIR /app

COPY --from=build /wheels /wheels
COPY requirements.txt .
RUN pip install --no-cache-dir --no-index --find-links=/wheels -r requirements.txt \
    && rm -rf /wheels

# Source last: editing code must not invalidate the dependency layer.
COPY alembic.ini ./
COPY alembic ./alembic
COPY app ./app

RUN useradd --no-create-home --shell /usr/sbin/nologin app \
    && chown -R app:app /app
USER app

EXPOSE 8000

# Gunicorn supervises a pool of Uvicorn workers: Uvicorn speaks ASGI,
# Gunicorn restarts a worker that dies without taking the container with it.
# The `worker` service in docker-compose overrides this with the Celery
# command; both run from this one image so their code cannot drift apart.
CMD ["sh", "-c", "gunicorn app.main:app --worker-class uvicorn.workers.UvicornWorker --workers ${WEB_CONCURRENCY} --bind 0.0.0.0:${PORT} --access-logfile - --error-logfile -"]
