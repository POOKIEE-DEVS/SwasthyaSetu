# SwasthyaSetu: project context for Claude Code

## Mode: shipping a hackathon demo

We are building a ~5-minute live demo, due in 4 days. The goal is **one
reliable end-to-end journey**, not feature breadth:

1. A patient chats with MedGemma, in English or Nepali.
2. The patient requests a doctor.
3. A volunteer doctor on another laptop accepts.
4. They talk on a live WebRTC video call.

Prefer small, working, verified changes. Anything that isn't on that
journey waits until after the demo.

## Stack

- **Frontend:** Next.js as a static export (TypeScript, Tailwind CSS v4,
  shadcn/ui, Zustand). This Next.js version has breaking changes: read
  `frontend/AGENTS.md` and `frontend/node_modules/next/dist/docs/` before
  using unfamiliar APIs.
- **Backend:** Python, FastAPI, Uvicorn. It serves the API, the WebSockets,
  and the built frontend from **one origin**.
- **Model:** `google/medgemma-1.5-4b-it` on a Hugging Face Gradio Space
  (`model-space/`), called via `gradio_client`.
- **Calls:** WebRTC. The backend relays signalling; Cloudflare provides TURN.
- **Deploy:** one Docker image (`Dockerfile`) on Render (`render.yaml`).

Architecture: `docs/architecture.md`. Deploy steps: `docs/deployment.md`.
Demo script: `docs/demo.md`. Requirements of record:
`docs/SwasthyaSetu-Phase3-Requirements.pdf`.

## Rules

- **Run the backend as exactly one process.** Consultations and call rooms
  live in memory. `--workers` or a second replica silently breaks calls.
- **No secrets in the repo.** `HF_TOKEN` and the TURN keys go in
  `backend/.env` (gitignored) or the Render and Space dashboards.
- **Keep the emergency path independent of the AI.** The 102 number and the
  urgent banner never wait on the model.
- **The model gives first-aid information, not diagnosis.** Don't add
  medicine doses or diagnostic claims to prompts or UI.
- **Verify before claiming something works:**
  - backend: `ruff check` + `pytest`
  - frontend: `lint` + `typecheck` + `build`
  - anything touching chat or calls: `scripts/smoke_test.py` against a
    running server
- Commit per logical change.

## Ship list

- [x] Fix config crash, single worker, same-origin frontend (no build-time API URL)
- [x] Chat endpoint → MedGemma Space: timeout, clear errors, history cap, rate limit
- [x] Chat page: bilingual, "thinking…" state, retry, emergency banner, survives refresh
- [x] Talk to a doctor: request → live doctor queue (tone alert) → atomic accept, chat handoff with consent
- [x] Video call: camera/mic, two-way audio+video, mute, camera off, hang up, voice-only fallback, refresh-rejoin
- [x] HF Space code (`model-space/`), Docker image, Render blueprint, CI
- [x] Two-browser end-to-end smoke test (`scripts/smoke_test.py`)
- [ ] Deploy the Space and accept the MedGemma terms (docs/deployment.md §1)
- [ ] Create Cloudflare TURN keys (§2)
- [ ] Deploy to Render; `/health` shows model and TURN configured (§3)
- [ ] Smoke test passes against the deployed URL
- [ ] Real call between two laptops on **different networks**
- [ ] Check real MedGemma replies to the demo sentences, in English and Nepali
- [ ] Rehearse the 5-minute script (docs/demo.md) and record a backup video

## Commands

```bash
cd backend && .venv/Scripts/pytest && .venv/Scripts/ruff check app tests ../model-space ../scripts
cd frontend && npm run lint && npm run typecheck && npm run build
python scripts/smoke_test.py http://localhost:8000     # needs `pip install playwright`
```
