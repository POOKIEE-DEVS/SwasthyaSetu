# SwasthyaSetu: one image, one process, one origin.
#
#   Stage 1 builds the Next.js static export.
#   Stage 2 is the FastAPI backend, which serves that export at "/" alongside
#   the API (/api/v1) and WebSockets (/ws).
#
# Must run as a SINGLE uvicorn process: consultations and call rooms live in
# memory, so a second worker would split them and calls would not connect.

FROM node:22-alpine AS web
WORKDIR /web
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
# NEXT_PUBLIC_API_URL stays unset: the browser talks to the same origin.
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM python:3.12-slim
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PORT=8000 \
    ENVIRONMENT=production \
    STATIC_DIR=/app/static \
    CORS_ORIGINS=""
WORKDIR /app

COPY backend/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY backend/app ./app
COPY --from=web /web/out ./static

RUN useradd --no-create-home --shell /usr/sbin/nologin app && chown -R app:app /app
USER app

EXPOSE 8000
# --proxy-headers: trust the platform's X-Forwarded-For so per-client rate
# limiting sees real client IPs, not the load balancer's.
CMD ["sh", "-c", "exec uvicorn app.main:app --host 0.0.0.0 --port ${PORT} --proxy-headers --forwarded-allow-ips='*'"]
