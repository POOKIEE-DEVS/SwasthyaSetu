# SwasthyaSetu — 20-Week Roadmap

This file mirrors the roadmap in [`CLAUDE.md`](../CLAUDE.md) and is kept in sync as
weeks complete. Each week is one phase; phases are grouped into five stages so the shape
of the whole project is easy to see at a glance.

Stack: Next.js PWA (TypeScript, Tailwind, shadcn/ui, Zustand) → FastAPI backend
(PostgreSQL, Redis, Celery, WebSockets, WebRTC) → AI engine (MedGemma, whisper.cpp,
Nepali TTS). See [`architecture.md`](architecture.md) for the full design.

## Stage A — Foundations (Weeks 1–4)

- [x] **Week 1 — Project Setup & Architecture Finalization**
  Repo scaffolding for frontend and backend, Docker Compose stack (backend, worker,
  frontend, postgres, redis, coturn), environment config, DB schema written as Alembic
  revisions (not yet applied), CI skeleton.
- [ ] **Week 2 — Database Implementation**
  Apply migrations, seed script with sample data, async session layer wired into
  request handlers, readiness endpoint reporting real Postgres and Redis health.
- [ ] **Week 3 — Auth: Core**
  User model, registration endpoint, argon2 password hashing, login endpoint issuing a
  JWT.
- [ ] **Week 4 — Auth: RBAC & Refresh Tokens**
  Refresh token rotation backed by Redis, role dependencies (Patient / Doctor / Admin),
  auth test suite.

## Stage B — API Surface & Frontend Foundations (Weeks 5–8)

- [ ] **Week 5 — API Hardening & Middleware**
  JWT verification dependency, Redis-backed rate limiting on auth and AI endpoints,
  WebSocket handshake authentication, consistent error responses.
- [ ] **Week 6 — Frontend Foundations**
  App Router routing, role-based layout shells, login/register pages wired to the auth
  endpoints, Zustand session store, authenticated fetch client.
- [ ] **Week 7 — Doctor Matching: Data Layer**
  `doctor_profiles` CRUD endpoints, doctor availability toggle with Redis-backed
  presence.
- [ ] **Week 8 — Doctor Matching: Matching Logic**
  Ranking pipeline (location → specialization → availability → language → experience),
  `POST /doctor/match`.

## Stage C — AI Triage Pipeline (Weeks 9–12)

- [ ] **Week 9 — AI Engine Skeleton**
  MedGemma client over HTTP (Ollama locally), `POST /triage` returning a real
  completion, inference dispatched to the Celery `ai` queue.
- [ ] **Week 10 — Symptom Extraction, Classification & Voice Input**
  Extract structured symptoms from free text; classify into Green / Yellow / Red; wire
  whisper.cpp so a Nepali voice note can be the input.
- [ ] **Week 11 — Knowledge Base & Retrieval**
  Curate first-aid articles (WHO, Red Cross, Nepal Health Protocol); PostgreSQL
  full-text retrieval; assemble retrieved articles and patient history into the prompt.
- [ ] **Week 12 — Safety Validator, Structured Response & Narration**
  Safety filter before responses reach the user; return `response`, `confidence`,
  `emergency_level`, `requires_doctor`, `suggested_action`; conversational memory per
  patient; Nepali text-to-speech narration of validated guidance.

## Stage D — Consultations & Product Surface (Weeks 13–16)

- [ ] **Week 13 — Consultations & WebRTC Calls**
  `consultations` wiring, signalling over `/ws/consultations/{id}`, time-limited TURN
  credentials, `POST /consultation` and `POST /call`.
- [ ] **Week 14 — Notification Service**
  Celery `notifications` queue with email / SMS / push delivery; triggers for
  doctor-accepted, call reminder, emergency alert, consultation finished; live push over
  the notification WebSocket.
- [ ] **Week 15 — Frontend Integration: Patient Flow**
  Triage UI (text and voice) → doctor matching UI → WebRTC call UI, wired end-to-end.
- [ ] **Week 16 — Frontend Integration: Doctor & Admin Flow**
  Doctor dashboard with the live queue (accept/reject consultations); admin panel
  (approve doctors, manage articles, view reports).

## Stage E — Hardening & Launch (Weeks 17–20)

- [ ] **Week 17 — Security Hardening**
  HTTPS and `wss://` everywhere, rate-limit tuning, audit logs, encryption at rest,
  security review pass.
- [ ] **Week 18 — Offline Emergency Mode**
  Service-worker caching of first-aid articles, offline detection UI, emergency
  contacts and GPS shown offline, background sync of queued symptom reports.
- [ ] **Week 19 — Testing & QA**
  End-to-end testing of triage, matching, consultation, and offline flows; bug fixing.
- [ ] **Week 20 — Deployment & Monitoring**
  Deploy frontend and backend to production hosting; structured logging finalized;
  Prometheus/Grafana dashboards; final demo prep.

---

## Completed weeks

### Week 1 — Project Setup & Architecture Finalization ✅

Delivered:

- **Monorepo** — `frontend/`, `backend/`, `docs/`, with the backend internally
  partitioned into `api/`, `services/`, `ai/`, `realtime/`, `models/`, `db/`, `workers/`.
- **Backend** — FastAPI app factory with structured request logging, request-id
  propagation, CORS, and gzip; liveness (`/health`) and readiness (`/health/ready`)
  endpoints; Celery app with separate `ai` and `notifications` queues; Redis client;
  async SQLAlchemy engine.
- **Real-time** — WebSocket connection manager with room fan-out, the three channels
  from architecture §7, and WebRTC ICE configuration with time-limited TURN credentials.
- **Frontend** — Next.js PWA: Tailwind CSS v4 with triage-level design tokens,
  shadcn/ui component layer, Zustand connectivity store, service worker with
  three caching strategies, web app manifest with generated icons, offline page.
- **Schema** — all eleven tables as SQLModel models and one Alembic initial revision,
  written but not yet applied.
- **Infrastructure** — Docker Compose covering backend, worker, frontend, postgres,
  redis, and coturn; `.env.example` per service; CI running lint, format, tests,
  typecheck, build, a full migrate/revert/re-migrate cycle, and compose validation.
