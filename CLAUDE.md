# SwasthyaSetu: project context for Claude Code

## Mode: shipping a hackathon demo

A ~5-minute live demo, due 2026-10-04. The goal is **one reliable
end-to-end journey** plus the trust layer around it:

1. A patient (no login) chats with MedGemma, in English or Nepali.
2. The patient requests a doctor.
3. A **verified** doctor, pharmacist or MBBS student accepts.
4. They talk on a live WebRTC video call; the patient sees "Verified Doctor".

Around it: Google sign-in, professional verification (KYC) reviewed by one
admin, and saved chats for signed-in patients.

Prefer small, working, verified changes. Anything else waits until after
the demo.

## Stack

- **Frontend:** Next.js as a static export (TypeScript, Tailwind CSS v4,
  shadcn/ui, Zustand). This Next.js version has breaking changes: read
  `frontend/AGENTS.md` and `frontend/node_modules/next/dist/docs/` before
  using unfamiliar APIs. `npm run build` also runs
  `scripts/flatten-segments.mjs` (fixes prefetch 404s in the export).
- **Backend:** Go (`backend/`: `cmd/server` + `internal/*`, standard
  library HTTP, `coder/websocket`). It serves the API, the WebSockets, and
  the built frontend from **one origin**. Package map:
  `docs/go-backend.md` (and `docs/architecture.md` §2).
- **Data:** Postgres (Neon) via `pgx`; SQLite (pure Go) locally. Tables are
  created at startup (`internal/database/schema.go`, `IF NOT EXISTS`, no
  migrations yet). Every SQL query lives in `internal/store`.
- **Auth:** Google OAuth (code + PKCE) done by the backend; HttpOnly session
  cookie, hashed in the database. `DEV_LOGIN=true` for local testing only.
- **Model:** `google/medgemma-1.5-4b-it` served by `model-space/app.py`
  (Gradio; the only non-Go, non-TypeScript code, because the model needs
  PyTorch), called over Gradio's HTTP API
  (`internal/ai/gradio.go`). Free: Colab T4 + a `gradio.live`
  link (`model-space/colab.ipynb`). Paid: a Hugging Face GPU Space.
- **Calls:** WebRTC. The backend relays signalling; TURN from ExpressTURN
  (free, static credentials) or Cloudflare.
- **Deploy:** one Docker image (`Dockerfile`) on Render (`render.yaml`).

Architecture: `docs/architecture.md`. Deploy steps: `docs/deployment.md`.
Demo script: `docs/demo.md`. Requirements of record:
`docs/SwasthyaSetu-Phase3-Requirements.pdf`.

## Rules

- **Run the backend as exactly one process.** The queue and call rooms live
  in memory. A second replica silently breaks calls.
- **No secrets in the repo.** `HF_TOKEN`, TURN keys, `DATABASE_URL` and the
  Google client secret go in `backend/.env` (gitignored) or the Render and
  Space dashboards.
- **Keep the emergency path independent of the AI, the login and the
  database.** The 102 number, the urgent banner, chat and "request a doctor"
  never wait on the model, never ask to sign in, and keep working if the
  database is down.
- **Only verified professionals see patients.** Check it on the server
  (`requireProfessional`, `verifiedProfessional` in `internal/api`), never
  only in the UI.
- **The model gives first-aid information, not diagnosis.** Don't add
  medicine doses or diagnostic claims to prompts or UI.
- **Commits:** one per logical change, with a detailed message (what, why,
  files, checks). No Claude co-author or attribution lines.
- **Verify before claiming something works:**
  - backend: `gofmt -l .` (no output) + `go vet ./...` + `go test -race ./...`
  - frontend: `lint` + `typecheck` + `build`
  - anything touching chat, calls, sign-in or verification:
    `go run ./cmd/smoketest <url>` (in `backend/`) against a running server

## Ship list

- [x] Fix config crash, single worker, same-origin frontend (no build-time API URL)
- [x] Chat endpoint → MedGemma: timeout, clear errors, history cap, rate limit
- [x] Chat page: bilingual, "thinking…" state, retry, emergency banner, survives refresh
- [x] Talk to a doctor: request → live queue (tone alert) → atomic accept, chat handoff with consent
- [x] Video call: camera/mic, two-way audio+video, mute, camera off, hang up, voice-only fallback, refresh-rejoin
- [x] Model hosting (Colab notebook + share link), Docker image, Render blueprint, CI
- [x] Deployed: model on Colab, ExpressTURN, Render; smoke test passed against the live URL
- [x] Database, Google sign-in, roles
- [x] Professional verification (doctor NMC / pharmacist NPC / student recommendation + citizenship), admin review page
- [x] Queue and accept limited to verified professionals; "Verified Doctor" badge for patients
- [x] Emergency-first home page; saved chats for signed-in patients
- [x] Smoke test covers apply → approve → call → badge → saved chats (13/13 locally)
- [ ] Create the Neon database and Google OAuth client; add the 4 new Render variables (docs/deployment.md §3–5)
- [ ] Approve the demo doctor's account; leave one sample application pending (§6)
- [ ] Smoke test against the deployed URL with `--pro-session` (§7)
- [ ] Real call between two laptops on **different networks**
- [ ] Check real MedGemma replies to the demo sentences, in English and Nepali
- [ ] Rehearse the 5-minute script (docs/demo.md) and record a backup video
- [x] Backend rewritten in Go: same API, same database tables, smoke test 14/14 locally
- [ ] After the Go deploy: check `/health`, then the smoke test against the live URL (§7)

## Commands

```bash
cd backend && gofmt -l . && go vet ./... && go test -race ./...
TEST_DATABASE_URL=postgres://... go test ./internal/store/ ./internal/api/   # optional, real Postgres
cd frontend && npm run lint && npm run typecheck && npm run build
# Local end-to-end (`go run ./cmd/server` with DEV_LOGIN=true ADMIN_EMAILS=admin@smoke.test):
cd backend && go run ./cmd/smoketest http://localhost:8000   # needs Chrome
```
