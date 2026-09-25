# SwasthyaSetu — Project Context for Claude Code

This file is the source of truth for building SwasthyaSetu. Read this fully before writing any code.

## What we're building

SwasthyaSetu ("the bridge to health") is a web-based healthcare access platform for
communities where qualified medical help is far away, unreliable, or out of reach. It
connects patients to volunteering doctors in emergencies, gives instant AI-assisted
first-aid guidance through a triage pipeline, and stays useful offline. Full product and
architecture detail lives in `docs/architecture.md`.

## Tech stack

### Frontend

| Concern | Technology |
|---|---|
| Core framework | Next.js (React), configured as a Progressive Web App (PWA) |
| Language & styling | TypeScript, Tailwind CSS, shadcn/ui |
| State management | Zustand (lightweight client-side state) |

### Backend

| Concern | Technology |
|---|---|
| Language & framework | Python, FastAPI |
| ASGI server | Uvicorn / Gunicorn |
| Database & ORM | PostgreSQL, SQLModel / SQLAlchemy, Alembic migrations |
| Validation & async processing | Pydantic, Celery (background tasks), Redis (cache, sessions, Celery queue) |

### Real-time & communication

| Concern | Technology |
|---|---|
| WebSockets | Native FastAPI WebSockets for live queues and notification broadcasts |
| Teleconsultation | WebRTC backed by TURN/STUN servers (coturn) for low-latency audio/video |

### AI & Edge ML engine

| Concern | Technology |
|---|---|
| Clinical triage model | MedGemma (base medical LLM) |
| Speech-to-text | whisper.cpp (optimized for on-device Nepali voice input) |
| Text-to-speech | Localized Nepali TTS engine for audio narration |

**Why one Python backend:** the triage pipeline needs the model, the patient's history,
the knowledge base, and consultation records in one place — putting it behind its own
network boundary turns every triage call into an extra hop plus duplicated models on
both sides.
The load is I/O-bound (queries, WebSocket frames, waiting on model calls), which async
FastAPI handles well at this scale; genuinely expensive CPU work is offloaded to Celery
workers. And for seven people over twenty weeks, one language means one test setup, one
dependency file, and one deployment story. The tradeoff — harder selective scaling — is
mitigated by the package boundaries in `app/services/` and `app/ai/`, which are where a
split would happen if it were ever needed.

## Repository structure

```
swasthyasetu/
├── backend/                 # Python + FastAPI — the entire backend
│   ├── app/
│   │   ├── main.py          # app factory, middleware, lifespan
│   │   ├── core/            # config, logging, security, celery app
│   │   ├── db/              # engines, sessions, redis, model registry
│   │   ├── models/          # SQLModel tables (the eleven of §6)
│   │   ├── schemas/         # Pydantic request/response shapes
│   │   ├── api/v1/          # routers
│   │   ├── services/        # auth, matching, consultation, notification, admin
│   │   ├── realtime/        # websockets, connection manager, WebRTC signalling
│   │   ├── ai/              # triage pipeline, llm (MedGemma), stt, tts, knowledge_base
│   │   └── workers/         # celery tasks
│   ├── alembic/versions/    # schema migrations
│   ├── tests/
│   └── requirements.txt
├── frontend/                # Next.js PWA — patient / doctor / admin views
│   ├── app/                 # App Router pages
│   ├── components/ui/       # shadcn/ui component layer
│   ├── lib/store/           # Zustand stores
│   ├── public/              # service worker, manifest, icons
│   └── package.json
├── docs/
│   ├── SwasthyaSetu-Phase3-Requirements.pdf   # the requirements of record
│   ├── architecture.md
│   └── roadmap.md           # this 20-week plan, kept in sync as we go
├── docker-compose.yml
└── CLAUDE.md                # this file
```

## Working agreement

- **Only build what the current week's checklist below asks for.** Do not jump ahead to
  future weeks even if it seems efficient — the point of this plan is visible, weekly
  progress for mentor review.
- When a task is completed, check it off in this file (`- [x]`) so the file stays an
  accurate log of progress.
- Every week should end in something runnable — even if it's just one service with a
  `/health` endpoint. Prefer a small working slice over a large half-finished one.
- Write commits per logical task, not one giant commit per week.
- Keep secrets out of the repo — use `.env` files (gitignored) with `.env.example`
  checked in.

---

## 20-Week Roadmap

Each week is one phase. Phases are grouped into five stages so the shape of the whole
project is easy to see at a glance.

### Stage A — Foundations (Weeks 1–4)

- [x] **Week 1 — Project Setup & Architecture Finalization**
  Repo scaffolding for frontend and backend, Docker Compose stack, environment config,
  DB schema written as Alembic revisions (not yet applied), CI skeleton.
- [ ] **Week 2 — Database Implementation**
  Apply migrations, seed script with sample data, async session layer wired into request
  handlers, readiness endpoint reporting real Postgres and Redis health.
- [ ] **Week 3 — Auth: Core**
  User model, registration endpoint, argon2 password hashing, login endpoint issuing a
  JWT.
- [ ] **Week 4 — Auth: RBAC & Refresh Tokens**
  Refresh token rotation backed by Redis, role dependencies (Patient / Doctor / Admin),
  auth test suite.

### Stage B — API Surface & Frontend Foundations (Weeks 5–8)

- [ ] **Week 5 — API Hardening & Middleware**
  JWT verification dependency, Redis-backed rate limiting, WebSocket handshake
  authentication, consistent error responses.
- [ ] **Week 6 — Frontend Foundations**
  App Router routing, role-based layout shells, login/register pages wired to the auth
  endpoints, Zustand session store, authenticated fetch client.
- [ ] **Week 7 — Doctor Matching: Data Layer**
  `doctor_profiles` CRUD endpoints, doctor availability toggle with Redis-backed
  presence.
- [ ] **Week 8 — Doctor Matching: Matching Logic**
  Ranking pipeline (location → specialization → availability → language → experience),
  `POST /doctor/match`.

### Stage C — AI Triage Pipeline (Weeks 9–12)

- [ ] **Week 9 — AI Engine Skeleton**
  MedGemma client over HTTP (Ollama locally), `POST /triage` returning a real completion,
  inference dispatched to the Celery `ai` queue.
- [ ] **Week 10 — Symptom Extraction, Classification & Voice Input**
  Extract structured symptoms from free text; classify into Green / Yellow / Red; wire
  whisper.cpp so a Nepali voice note can be the input.
- [ ] **Week 11 — Knowledge Base & Retrieval**
  Curate first-aid articles (WHO, Red Cross, Nepal Health Protocol); PostgreSQL full-text
  retrieval; assemble retrieved articles and patient history into the prompt.
- [ ] **Week 12 — Safety Validator, Structured Response & Narration**
  Safety filter before responses reach the user; return `response`, `confidence`,
  `emergency_level`, `requires_doctor`, `suggested_action`; conversational memory per
  patient; Nepali text-to-speech narration of validated guidance.

### Stage D — Consultations & Product Surface (Weeks 13–16)

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

### Stage E — Hardening & Launch (Weeks 17–20)

- [ ] **Week 17 — Security Hardening**
  HTTPS and `wss://` everywhere, rate-limit tuning, audit logs, encryption at rest,
  security review pass.
- [ ] **Week 18 — Offline Emergency Mode**
  Service-worker caching of first-aid articles, offline detection UI, emergency contacts
  and GPS shown offline, background sync of queued symptom reports.
- [ ] **Week 19 — Testing & QA**
  End-to-end testing of triage, matching, consultation, and offline flows; bug fixing.
- [ ] **Week 20 — Deployment & Monitoring**
  Deploy frontend and backend to production hosting; structured logging finalized;
  Prometheus/Grafana dashboards; final demo prep.

---

## ✅ Week 1 complete — awaiting go-ahead for Week 2

Week 1 is done and checked off; `docs/roadmap.md` records what shipped. **Do not start
Week 2 work until given the go-ahead.**

### What Week 1 delivered

- [x] Monorepo initialized (`backend/`, `frontend/`, `docs/`).
- [x] **backend/**: FastAPI app with `GET /health` (liveness) and `GET /health/ready`
      (Postgres + Redis), structured request logging with request-id propagation,
      Celery app with `ai` and `notifications` queues, async SQLAlchemy engine, Redis
      client.
- [x] **backend/**: WebSocket connection manager with room fan-out; the three channels
      from architecture §7; WebRTC ICE configuration with time-limited TURN credentials.
- [x] **frontend/**: Next.js (TypeScript) PWA — Tailwind CSS v4 with triage-level design
      tokens, shadcn/ui component layer, Zustand connectivity store, service worker,
      web app manifest with generated icons, offline page, placeholder home page.
- [x] `docker-compose.yml` covering `backend`, `worker`, `frontend`, `postgres`,
      `redis`, and `coturn`.
- [x] `.env.example` per service, real `.env` files gitignored.
- [x] All eleven tables as SQLModel models plus one Alembic initial revision — written
      and verified, applied in Week 2.
- [x] GitHub Actions CI: backend lint/format/test, a full migrate → revert → re-migrate
      cycle with a model/migration drift check, frontend lint/typecheck/build, and
      compose validation.
- [x] Top-level `README.md`.

### Verifying the week's definition of done

```bash
docker compose up --build          # backend, worker, frontend, postgres, redis, coturn
curl localhost:8000/health         # {"status":"ok","service":"backend",...}
curl localhost:8000/health/ready   # postgres + redis dependency report
open http://localhost:3000         # placeholder page loads

cd backend && pytest && ruff check app tests alembic
cd frontend && npm run lint && npm run typecheck && npm run build
```
