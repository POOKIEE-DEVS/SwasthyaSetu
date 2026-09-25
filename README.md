# SwasthyaSetu

*The bridge to health* · Built by team **Pookiedevs** for the Student Partnership Program.

**SwasthyaSetu** is a healthcare-access platform for communities where qualified medical
help is far away, unreliable, or out of reach. It connects patients to volunteering
doctors through WebRTC video consultation, provides AI-assisted first-aid triage grounded
in curated medical sources, and stays useful offline — because the moments when guidance
matters most are often the moments connectivity is weakest.

- **System design, data model, and API shape:** [docs/architecture.md](docs/architecture.md)
- **20-week roadmap and weekly progress:** [docs/roadmap.md](docs/roadmap.md) · working
  agreement and active week in [CLAUDE.md](CLAUDE.md)
- **Phase-1 proposal and meeting minutes:**
  [POOKIEE-DEVS/SwasthyaSetu-Phase-1](https://github.com/POOKIEE-DEVS/SwasthyaSetu-Phase-1)

## Stack

| Layer | Technology |
|---|---|
| Frontend | Next.js (React) as a PWA · TypeScript · Tailwind CSS · shadcn/ui · Zustand |
| Backend | Python · FastAPI · Uvicorn / Gunicorn |
| Data | PostgreSQL · SQLModel / SQLAlchemy · Alembic · Redis |
| Async | Celery (background tasks), Redis as broker |
| Real-time | Native FastAPI WebSockets · WebRTC over STUN/TURN (coturn) |
| AI & Edge ML | MedGemma (clinical triage) · whisper.cpp (Nepali STT) · Nepali TTS |

## Running the stack

Prerequisite: [Docker Desktop](https://www.docker.com/products/docker-desktop/) (or any
Docker Engine with Compose v2).

```bash
docker compose up --build
```

| Service | URL | What you should see |
|---|---|---|
| frontend | http://localhost:3000 | Placeholder home page (installable as a PWA) |
| backend | http://localhost:8000/health | `{"status":"ok","service":"backend",...}` |
| backend | http://localhost:8000/health/ready | Postgres + Redis dependency report |
| backend | http://localhost:8000/docs | Interactive OpenAPI docs |
| worker | — | Celery worker consuming the `ai` and `notifications` queues |
| postgres | localhost:5432 | Accepts connections (schema applied in Week 2) |
| redis | localhost:6379 | Cache, sessions, and Celery broker |
| coturn | localhost:3478 | STUN/TURN for WebRTC teleconsultation |

Configuration is environment-driven: compose runs with safe dev defaults out of the box,
and each service documents its variables in its own `.env.example` (copy to `.env` to
override — real `.env` files are gitignored).

### Running without Docker

```bash
# Backend — needs PostgreSQL and Redis reachable at the URLs in backend/.env
cd backend
python -m venv .venv && .venv/bin/pip install -r requirements-dev.txt
.venv/bin/uvicorn app.main:app --reload
.venv/bin/celery -A app.core.celery_app:celery_app worker --loglevel=info

# Frontend
cd frontend && npm install && npm run dev
```

### Database migrations

The schema is written but not yet applied — that lands in Week 2.

```bash
docker compose exec backend alembic upgrade head     # apply
docker compose exec backend alembic downgrade base   # revert
cd backend && alembic upgrade head --sql             # inspect the DDL, no database needed
```

### Tests and checks

`make check` runs everything CI runs; `make help` lists every target.

```bash
make check                 # backend lint + tests, frontend lint/typecheck/build, compose

# or directly:
cd backend  && ruff check app tests alembic && ruff format --check app tests && pytest
cd frontend && npm run lint && npm run typecheck && npm run build
```

## Repository layout

| Path | Contents |
|---|---|
| [backend/](backend/) | FastAPI — API, auth, matching, consultations, notifications, admin, AI engine, workers |
| [backend/app/ai/](backend/app/ai/) | Triage pipeline, MedGemma client, whisper.cpp STT, Nepali TTS, knowledge base |
| [backend/app/realtime/](backend/app/realtime/) | WebSocket channels and WebRTC signalling |
| [backend/alembic/versions/](backend/alembic/versions/) | Versioned PostgreSQL schema |
| [frontend/](frontend/) | Next.js PWA (TypeScript) — patient, doctor, and admin views |
| [docs/](docs/) | Architecture and roadmap |
| [.github/workflows/](.github/workflows/) | CI — lint, test, migrations, build |
| [Makefile](Makefile) | Dev commands — `make help` |

## Team

| Name | Responsibility |
|------|----------------|
| Aaditya Raj Uprety | Project Lead & Backend Development |
| Arekh Shrestha | Frontend Development (Mobile App) |
| Raksha Karn | AI/ML Engineer (Edge & Cloud Models) |
| Shreyam Regmi | Backend Integration |
| Smriti Adhikari | UI/UX Design & Frontend Website |
| Srijit Gyawali | Full Stack |
| Yojana Ghimire | AI/ML Engineer (Edge & Cloud Models) |
