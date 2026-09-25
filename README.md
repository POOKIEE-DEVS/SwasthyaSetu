# SwasthyaSetu

*The bridge to health* · स्वास्थ्य सेतु · Built by team **Pookiedevs**.

First-aid guidance from an AI assistant, and a live video call with a volunteer
doctor, for communities where medical help is far away.

**The demo flow:**

1. A patient describes what's happening, in **English or Nepali**. **MedGemma**
   replies with first-aid steps. Emergencies immediately show *Call 102* and
   *Talk to a doctor*.
2. The patient requests a doctor and can share the chat, so they don't have to
   repeat themselves.
3. A volunteer doctor, online on another laptop, is alerted, reads the chat,
   and accepts.
4. They talk on a live **WebRTC video call**, with mute, camera off, and hang
   up. If there's no camera, the call continues as voice-only.

## Stack

| Layer | Technology |
|---|---|
| Frontend | Next.js (static export, installable PWA) · TypeScript · Tailwind CSS · shadcn/ui · Zustand |
| Backend | Python · FastAPI · Uvicorn (serves the API, WebSockets, and the frontend from one origin) |
| Real-time | Native FastAPI WebSockets · WebRTC with STUN/TURN (Cloudflare) |
| AI | MedGemma 1.5 4B (`google/medgemma-1.5-4b-it`) on a Hugging Face GPU Space |

How it fits together: [docs/architecture.md](docs/architecture.md).

## Run it locally

```bash
# Backend (API on :8000)
cd backend
python -m venv .venv
.venv/Scripts/pip install -r requirements-dev.txt     # macOS/Linux: .venv/bin/pip
cp .env.example .env                                  # set HF_SPACE_ID and HF_TOKEN
.venv/Scripts/uvicorn app.main:app --reload

# Frontend (on :3000), in a second terminal
cd frontend
cp .env.example .env.local
npm install
npm run dev
```

Open <http://localhost:3000>. Use two browser windows, one as the patient and
one as the doctor. Or run the production image with `docker compose up --build`
and open <http://localhost:8000>.

## Deploy and demo

- **[docs/deployment.md](docs/deployment.md)**: MedGemma Space → Cloudflare
  TURN → Render. About an hour the first time.
- **[docs/demo.md](docs/demo.md)**: the 5-minute script, pre-demo checklist,
  and what to do if something fails on stage.
- **Before every rehearsal:** `python scripts/smoke_test.py <your-url>` runs
  the whole flow in two real Chrome windows and reports each step.

## Checks

```bash
cd backend  && .venv/Scripts/ruff check app tests ../model-space ../scripts && .venv/Scripts/pytest
cd frontend && npm run lint && npm run typecheck && npm run build
```

`make help` lists shortcuts (on Windows, run `make` from Git Bash). CI also
builds the Docker image and checks that it boots and serves the pages.

## Repository layout

| Path | Contents |
|---|---|
| [backend/](backend/) | FastAPI app: chat, consultations, WebSocket signalling, MedGemma client |
| [frontend/](frontend/) | Next.js app: patient and doctor pages, chat, video call |
| [model-space/](model-space/) | The Hugging Face Space that serves MedGemma |
| [scripts/smoke_test.py](scripts/smoke_test.py) | Two-browser end-to-end check of the demo |
| [docs/](docs/) | Architecture, deployment, demo playbook, Phase III requirements |
| [Dockerfile](Dockerfile) · [render.yaml](render.yaml) | One image; one Render service |

## Important

SwasthyaSetu gives **first-aid information, not a medical diagnosis**. In an
emergency in Nepal, call **102**. This is a hackathon demo: it has no user
accounts yet, and the doctor side is open to anyone with the link.

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
