# SwasthyaSetu: one image, one process, one origin.
#
#   Stage 1 builds the Next.js static export.
#   Stage 2 builds the Go backend: one static binary, no runtime needed.
#   Stage 3 runs that binary, which serves the export at "/" alongside the
#   API (/api/v1) and WebSockets (/ws). The base image has no shell or
#   package manager, and the server runs as a non-root user.
#
# Must run as a SINGLE process: consultations and call rooms live in memory,
# so a second instance would split them and calls would not connect.

FROM node:22-alpine AS web
WORKDIR /web
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
# NEXT_PUBLIC_API_URL stays unset: the browser talks to the same origin.
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM golang:1.26-alpine AS api
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# CGO off: the SQLite driver is pure Go, so the binary needs no C library.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# Includes CA certificates (HTTPS to Neon, Google, Hugging Face, Cloudflare).
FROM gcr.io/distroless/static-debian12:nonroot
ENV PORT=8000 \
    ENVIRONMENT=production \
    STATIC_DIR=/app/static \
    CORS_ORIGINS=""
WORKDIR /app
COPY --from=api /out/server /app/server
COPY --from=web /web/out /app/static
USER nonroot:nonroot
EXPOSE 8000
# No curl in the image: the binary checks its own /health.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/app/server", "-healthcheck"]
ENTRYPOINT ["/app/server"]
