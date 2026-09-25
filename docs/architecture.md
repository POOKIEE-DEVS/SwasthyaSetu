# SwasthyaSetu — Technical Architecture

*A web-based healthcare access platform with AI-assisted triage.*
Team: Pookie Devs · September 2026

> This document is the source of truth for system design, the data model, and the
> API shape. The week-by-week plan that delivers it lives in
> [`docs/roadmap.md`](roadmap.md).

## 1. Project Overview

SwasthyaSetu ("the bridge to health") is a healthcare access platform for communities
where qualified medical help is far away, unreliable, or out of reach. It connects
patients to volunteering doctors in emergencies, delivers trustworthy first-aid guidance
through an AI triage pipeline, and works even without a reliable internet connection —
because the moments when guidance matters most are often the moments connectivity is
weakest.

Who it serves:

- **Everyday citizens** — turn to the app when a family member is injured or suddenly ill.
- **Caregivers & first responders** — need immediate, step-by-step instructions during an
  emergency.
- **Volunteer doctors** — offer their time for emergency consultations through the
  platform.
- **Administrators** — verify doctors, monitor platform activity, and manage content.

## 2. System Architecture

A Next.js Progressive Web App talks to a single FastAPI backend, which owns
authentication, matching, consultations, notifications, and the AI triage pipeline.
Long-running work is pushed onto Celery workers so no request ever waits on model
inference. PostgreSQL holds all persistent state; Redis is cache, session store, and task
broker.

| Layer | Technology | Responsibility |
|---|---|---|
| Web app | Next.js (React), installable as a PWA | Role-based views for patients, doctors, and admins in a single application; offline shell and cached first-aid content. |
| Language & styling | TypeScript, Tailwind CSS, shadcn/ui | Typed UI code and a component layer the whole team shares. |
| Client state | Zustand | Lightweight client-side state — connectivity, session, active consultation. |
| API | Python + FastAPI | Every HTTP and WebSocket endpoint; request validation via Pydantic. |
| ASGI server | Uvicorn workers under Gunicorn | Uvicorn speaks ASGI; Gunicorn supervises the worker pool. |
| Persistence | PostgreSQL with SQLModel / SQLAlchemy | All persistent data — see §6. Schema versioned with Alembic. |
| Cache & queue | Redis | Response cache, sessions, rate-limit counters, and the Celery broker. |
| Background work | Celery | Model inference, transcription, speech synthesis, notification delivery. |
| Real-time events | Native FastAPI WebSockets | Consultation signalling, notification push, live doctor queue. |
| Teleconsultation | WebRTC over STUN/TURN (coturn) | Peer-to-peer audio and video; media never transits the server. |
| Clinical model | MedGemma | Triage reasoning, reached over HTTP. |
| Speech-to-text | whisper.cpp | On-device Nepali voice input. |
| Text-to-speech | Localized Nepali TTS engine | Audio narration of validated guidance. |

### Why a single Python backend

The backend is one FastAPI application rather than a fleet of small services. The
reasoning is specific to this project rather than general:

- **The AI pipeline is the hard part, and it is Python.** Triage needs the model, the
  patient's history, the knowledge base, and the consultation records together in one
  place. Putting the pipeline behind its own network boundary would turn every triage
  call into an extra hop plus a duplicated set of models on both sides of it.
- **Concurrency here is I/O-bound, not CPU-bound.** The load is database queries,
  WebSocket frames, and waiting on model calls. Async FastAPI handles that shape well at
  this platform's scale; the heavy CPU work is offloaded to Celery workers, which is
  where a separate process actually buys something.
- **Seven people, twenty weeks.** One language means one test setup, one dependency file,
  one deployment story, and no context-switch tax on a student team building part-time.

The tradeoff is real and worth stating: a single application is easier to build and
harder to scale selectively. Should one part of the system need to scale independently
later, the internal package boundaries in `app/services/` and `app/ai/` are where it
would be split, and Celery queues already isolate the expensive work.

## 3. AI Triage Pipeline

The AI engine does more than pass a question to a language model. Every query moves
through a pipeline that extracts symptoms, classifies urgency, grounds the answer in real
medical content, and validates the response before it ever reaches the user. Each stage
narrows what the model is allowed to be wrong about.

| Stage | Module | Week | What it does |
|---|---|---|---|
| 1 | `ai.triage.extraction` | 10 | Free text (or a whisper.cpp transcript) → structured symptoms |
| 2 | `ai.triage.classification` | 10 | Structured symptoms → Green / Yellow / Red |
| 3 | `ai.triage.retrieval` | 11 | Pull matching first-aid articles from the knowledge base |
| 4 | `ai.triage.prompt` | 11 | Assemble symptoms + retrieved articles + patient history |
| 5 | `ai.triage.inference` | 9 | Call MedGemma via `ai.llm` |
| 6 | `ai.triage.safety` | 12 | Validate before anything is shown |

### The models

**MedGemma** is the clinical triage model — medically tuned rather than general purpose,
because triage answers must be grounded in clinical language, not plausible-sounding
prose. It is reached over HTTP so the same code path serves an Ollama container in
development and a hosted endpoint in production. Temperature defaults to 0: the same
described symptoms must not produce different medical guidance on two consecutive runs.

**whisper.cpp** handles Nepali speech-to-text, on-device rather than through a cloud API,
for two reasons that both matter here: it works without connectivity, which is precisely
when a first-aid app is most needed; and audio of someone describing a medical emergency
never leaves the phone. Voice input is the point — a caregiver mid-emergency should be
able to describe what they see out loud, and many users are far more fluent speaking
Nepali than typing it.

**Nepali text-to-speech** narrates validated guidance. This is an accessibility
requirement, not a nicety: someone performing first aid has both hands occupied and
cannot read a screen, and a meaningful share of the target users are more comfortable
listening than reading. Only safety-validated text is ever synthesized — audio is harder
to skim critically than text.

### Example response shape

The pipeline never returns free text. It returns a structured response the frontend can
act on directly:

| Field | Example value |
|---|---|
| `response` | "Based on your symptoms, this may be a cardiac issue. Sit upright, stay calm, and avoid exertion..." |
| `confidence` | 94% |
| `emergency_level` | `red` |
| `requires_doctor` | `true` |
| `suggested_action` | Call Volunteer Doctor |

### Safety validator

Every model-generated response passes through a safety filter before reaching the user.
It blocks unsafe or hallucinated instructions — specific medication dosages, for
instance — and falls back to a conservative, pre-approved response plus a doctor
recommendation whenever the output cannot be verified against the knowledge base.

The general rule across the pipeline: **if any stage cannot complete confidently, the
result is a conservative answer and a doctor recommendation, never a guess.** A wrong
"you are fine" is far more dangerous here than an unnecessary escalation.

### Conversational memory

Symptom reports are linked to a patient's history rather than treated as one-off queries:
if someone reports a fever today and difficulty breathing tomorrow, stage 4 pulls both
into the prompt rather than evaluating each message in isolation. This context lives in
`triage_logs` and `symptom_reports` (§6) and is scoped to a single patient — never shared
across accounts.

## 4. Emergency Detection Layer

Every triage result carries an emergency level, which determines what the app shows next:

| Level | Example | What the app does |
|---|---|---|
| Green | Mild headache, minor cuts | Shows AI first-aid guidance only. |
| Yellow | High fever, persistent pain | Shows AI guidance and recommends booking a doctor. |
| Red | Chest pain, heavy bleeding, unconsciousness | Immediately surfaces "Join Doctor Now", emergency contact numbers, and the nearest hospital. |

These three levels are design tokens in the frontend (`--triage-green` / `-yellow` /
`-red`), defined once so that a Red-level result can never be rendered in a shade that
reads as merely a warning.

## 5. Doctor Matching

Rather than assigning the first available doctor, matching ranks candidates through a
short pipeline:

1. **Location** — proximity to the patient, where relevant.
2. **Specialization** — matched against the triage symptom classification.
3. **Availability** — real-time online/offline status.
4. **Language** — matched to the patient's preferred language.
5. **Experience** — used as a tiebreaker among otherwise equal candidates.

The result is a single assigned doctor, notified immediately over their WebSocket channel
and through the notification queue.

## 6. Data Model

| Table | Purpose |
|---|---|
| `users` | Base account for every person — patient, doctor, or admin. |
| `doctor_profiles` | Specialty, license status, languages, and experience for doctor accounts. |
| `appointments` | Scheduled (non-emergency) consultations. |
| `consultations` | Emergency consultation records — participants, call room, status, timestamps. |
| `medical_history` | Longer-term patient context referenced by the AI engine. |
| `triage_logs` | Every AI triage interaction — symptoms, emergency level, confidence, whether it escalated. |
| `symptom_reports` | Individual symptom entries linked to a patient over time, feeding conversational memory. |
| `articles` | Curated first-aid and medical reference content (see §9). |
| `doctor_ratings` | Patient feedback after a consultation. |
| `notifications` | Delivery record for email, SMS, and push notifications. |
| `emergency_contacts` | Patient-defined contacts to alert during a Red-level emergency. |

Tables are declared as SQLModel classes in [`backend/app/models/`](../backend/app/models/)
and versioned as Alembic revisions in
[`backend/alembic/versions/`](../backend/alembic/versions/). CI applies every migration,
reverts it, re-applies it, and then asserts that the models and the migrations agree — a
migration that cannot be applied is a draft, not a schema.

## 7. API Design

REST endpoints, all under `/api/v1`:

| Endpoint | Purpose |
|---|---|
| `POST /auth/register`, `/auth/login` | Account creation and login; issues JWT + refresh token. |
| `POST /auth/refresh` | Exchange a refresh token for a new access token. |
| `POST /triage` | Runs the AI triage pipeline on a description of symptoms. |
| `POST /triage/voice` | Same, from an uploaded Nepali voice note (whisper.cpp). |
| `POST /chat` | Follow-up message within an ongoing triage conversation. |
| `POST /doctor/match` | Finds and assigns an available doctor. |
| `POST /consultation` | Creates a consultation record and a call room. |
| `POST /call` | Issues ICE servers and short-lived TURN credentials for a join. |
| `POST /emergency` | Triggers the Red-level emergency path directly. |
| `GET /history` | A patient's past consultations and triage logs. |
| `GET /articles` | First-aid reference content; cached for offline use. |
| `GET /reports`, `POST /doctor/approve` | Admin-only verification and reporting (§10). |

Health and readiness sit outside the version prefix, at `/health` and `/health/ready`, so
orchestrators need not know about API versioning. Liveness deliberately depends on
nothing external — restarting a container because Postgres blipped turns a recoverable
outage into an unrecoverable one.

### WebSocket channels

| Channel | Purpose |
|---|---|
| `/ws/consultations/{id}` | WebRTC signalling relay for one call. |
| `/ws/notifications/{user_id}` | Server-to-client push for one account. |
| `/ws/doctors/queue` | The live queue of emergency requests that on-call doctors watch. |

## 8. Real-Time & Teleconsultation

**WebSockets** carry application events — the doctor's live queue, notification
broadcasts, triage progress. FastAPI serves them natively, so there is no separate
message broker in the request path. Connections are tracked per room in
`app.realtime.connection_manager`; running more than one backend replica means fanning
broadcasts out over Redis pub/sub, and that manager is the single seam where that gets
added.

**WebRTC** carries the consultation's own audio and video, peer-to-peer. The server never
sees media; it only relays the SDP offer/answer and ICE candidates that let two peers find
each other. This keeps a video call cheap to serve and private by construction.

STUN suffices when both peers are directly reachable. **TURN relays media when they are
not**, which is not an edge case here — symmetric NAT and restrictive mobile carrier
networks are common in the areas SwasthyaSetu targets, so a TURN server (coturn) is a
requirement rather than an optimisation. TURN credentials are minted per join as
time-limited HMACs, so the static shared secret is never handed to a browser.

## 9. Medical Knowledge Base

The content the AI engine retrieves from is curated, not generated freeform, so guidance
stays traceable to a real source:

- WHO first-aid and emergency-care guidelines.
- Red Cross first-aid protocols.
- Nepal Health Protocol references.
- A curated emergency handbook maintained by the project's medical contributors.

`articles.source_name` is required and `source_url` is recorded, which is what makes a
response auditable after the fact.

Retrieval (Week 11) starts as PostgreSQL full-text search. That is a deliberate first
choice rather than a placeholder: it needs no extra infrastructure, the corpus is small
and hand-curated, and keyword matching is debuggable in a way embedding similarity is
not — when a clinician asks why an article surfaced, there is an answer. Embeddings
(`pgvector`) become worth their cost once the corpus is large enough that vocabulary
mismatch is the real failure mode.

## 10. Admin Panel & Analytics

### Admin panel

- Approve or reject new doctor applications.
- View platform-wide reports and flagged consultations.
- Monitor ongoing and past calls.
- Manage first-aid articles and knowledge base content.
- Suspend or ban accounts that violate platform policy.

### Analytics dashboard

- Daily active patients and doctors.
- Average AI response time and doctor response time.
- AI usage volume and emergency-level distribution (Green/Yellow/Red).
- Doctor availability and total consultation count over time.

## 11. Offline Emergency Mode

Connectivity is exactly what is unreliable in the communities SwasthyaSetu is built for,
so the app is a Progressive Web App and offline support is a product requirement rather
than progressive enhancement.

The service worker uses three strategies, chosen by what a stale copy costs:

| Resource | Strategy | Why |
|---|---|---|
| App shell | Cache-first | The UI does not change mid-session. |
| First-aid articles | Stale-while-revalidate | Yesterday's article instantly beats a spinner; it refreshes in the background. |
| Everything else | Network-first | Doctor availability and triage results must never be stale — an offline answer there would be actively unsafe. |

When a user opens the app with no signal, previously viewed first-aid articles, emergency
contact numbers, and their GPS location remain available. Symptom reports logged offline
are queued locally (`symptom_reports.reported_offline`) and synced when connectivity
returns.

## 12. Security

- HTTPS everywhere; `wss://` for every WebSocket in any deployed environment.
- Passwords hashed with argon2, never stored in plain text.
- Short-lived JWT access tokens paired with rotating refresh tokens.
- Role-based access control across three roles: Patient, Doctor, Admin.
- Rate limiting backed by Redis counters, on both the auth and AI endpoints.
- Audit logs for sensitive actions — doctor approval, account changes, data access.
- Encryption at rest for medical history and consultation data in PostgreSQL.
- TURN credentials are per-join and time-limited; the static secret never reaches a client.

## 13. Logging & Monitoring

Every request is tagged with a request id (honouring an inbound `X-Request-ID` so the id
stays stable across the frontend-to-backend hop) and logged with its method, path, status,
and duration. Logs are human-readable lines in development and one JSON object per line in
production, for ingestion by a log aggregator.

The monitoring plan follows the standard stack: Prometheus for metrics collection and
Grafana for dashboards, tracking latency, CPU, memory, queue depth, and health-check
status. The logging layer is built first; full metrics dashboards land once the core
platform is stable (Week 20).

## 14. Implementation Roadmap

The week-by-week execution plan lives in [`docs/roadmap.md`](roadmap.md); the active week
is tracked in [`CLAUDE.md`](../CLAUDE.md).
