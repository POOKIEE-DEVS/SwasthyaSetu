# SwasthyaSetu

*The bridge to health* · स्वास्थ्य सेतु · Built by team **Pookiedevs**.

First-aid guidance from an AI assistant, and a live video call with a
**verified** doctor, pharmacist or MBBS student, for communities where medical
help is far away.

**The demo flow:**

1. **Emergency first.** The app opens straight on the first-aid chat, with a
   102 ambulance button. No sign-up, no menu.
2. A patient describes what's happening, in **English or Nepali**. **MedGemma**
   replies with first-aid steps. Emergencies immediately show *Call 102* and
   *Talk to a professional*.
3. The patient requests a doctor (still no login) and can share the chat, so
   they don't have to repeat themselves.
4. A **verified** medical professional, online on another laptop, is alerted,
   reads the chat, and accepts.
5. They talk on a live **WebRTC video call**, with mute, camera off, and hang
   up. The patient sees **"Verified Doctor · <name>"**. If there's no camera,
   the call continues as voice-only.

**Trust and accounts:**

- **Google sign-in.** Optional for patients (it keeps their chats under "My
  chats"); required for professionals.
- **Professional verification (KYC).** Doctors submit their Nepal Medical
  Council number, pharmacists their Nepal Pharmacy Council number, and MBBS
  students a doctor's letter of recommendation; everyone uploads their
  citizenship certificate. An admin reviews every application by hand. Only
  approved professionals can see waiting patients or take calls.

## Stack

| Layer | Technology |
|---|---|
| Frontend | Next.js (static export, installable PWA) · TypeScript · Tailwind CSS · shadcn/ui · Zustand |
| Backend | Python · FastAPI · Uvicorn (serves the API, WebSockets, and the frontend from one origin) |
| Data | Postgres (Neon) via SQLModel; SQLite for local development |
| Auth | Google OAuth 2.0 (authorization code + PKCE), server-side sessions |
| Real-time | Native FastAPI WebSockets · WebRTC with STUN/TURN (ExpressTURN or Cloudflare) |
| AI | MedGemma 1.5 4B (`google/medgemma-1.5-4b-it`) on a Colab GPU or a Hugging Face GPU Space |

How it fits together: [docs/architecture.md](docs/architecture.md).

## Run it locally

```bash
# Backend (API on :8000)
cd backend
python -m venv .venv
.venv/Scripts/pip install -r requirements-dev.txt     # macOS/Linux: .venv/bin/pip
cp .env.example .env                                  # set HF_SPACE_ID; DEV_LOGIN=true
.venv/Scripts/uvicorn app.main:app --reload

# Frontend (on :3000), in a second terminal
cd frontend
cp .env.example .env.local
npm install
npm run dev
```

Open <http://localhost:3000>. Locally the database is a SQLite file, and
`DEV_LOGIN=true` adds a development sign-in (any email, no Google) so you can
be a patient, a professional, and the admin (an email in `ADMIN_EMAILS`) in
separate browser windows. Or run the production image with
`docker compose up --build` and open <http://localhost:8000>.

## Deploy and demo

- **[docs/deployment.md](docs/deployment.md)**: MedGemma (Colab) → TURN
  relay → Neon database → Google sign-in → Render → first admin and verified
  professional. About an hour and a half the first time.
- **[docs/demo.md](docs/demo.md)**: the 5-minute script, pre-demo checklist,
  and what to do if something fails on stage.
- **Before every rehearsal:** `python scripts/smoke_test.py <your-url>` runs
  the whole flow in real Chrome windows and reports each step. Locally it
  also covers applying, admin approval and saved chats.

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
| [backend/](backend/) | FastAPI app: chat, consultations, WebSocket signalling, MedGemma client, sign-in, verification, saved chats |
| [frontend/](frontend/) | Next.js app: chat, account, verification form, professional dashboard, admin review, video call |
| [model-space/](model-space/) | The Hugging Face Space that serves MedGemma |
| [scripts/smoke_test.py](scripts/smoke_test.py) | Multi-browser end-to-end check of the demo |
| [docs/](docs/) | Architecture, deployment, demo playbook, Phase III requirements |
| [Dockerfile](Dockerfile) · [render.yaml](render.yaml) | One image; one Render service |

## Important

SwasthyaSetu gives **first-aid information, not a medical diagnosis**. In an
emergency in Nepal, call **102**. This is a hackathon build: professional
verification is a manual document review, not a link to the Nepal Medical
Council or Pharmacy Council registers.

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
