# Deploying SwasthyaSetu

Five pieces, each on a platform suited to it. All have a free option.

| Piece | Where | Why there |
|---|---|---|
| MedGemma model | Google Colab GPU (free) or a Hugging Face GPU Space (paid) | Needs a GPU. Loads the gated model with your token. |
| TURN relay for video | ExpressTURN (free) or Cloudflare Realtime TURN | Calls between different networks need a relay. Render has no UDP. |
| Database | Neon (free Postgres) | Accounts, verification documents and saved chats must survive restarts. |
| Google sign-in | Google Cloud OAuth client (free) | Patients and professionals sign in with their Google account. |
| App (website + API + call signalling) | Render, one Docker web service | HTTPS (required for cameras) and WebSockets, from one origin. |

Do them in this order. Allow about an hour and a half the first time.

---

## 1. The MedGemma model

Both options run the same code, [`model-space/app.py`](../model-space/app.py).
Either way, first **accept the model terms**: signed in to Hugging Face, open
[google/medgemma-1.5-4b-it](https://huggingface.co/google/medgemma-1.5-4b-it)
and accept the Health AI Developer Foundations terms. Then create a **read**
token (Settings → Access Tokens). Keep it private: never in chat or git.

### Option A (free): Google Colab GPU

A free Hugging Face account can only create static Spaces, so the free route
is Colab's free T4 GPU, with Gradio publishing a public link.

1. Open [`model-space/colab.ipynb`](../model-space/colab.ipynb) in Colab:
   <https://colab.research.google.com/github/POOKIEE-DEVS/SwasthyaSetu/blob/main/model-space/colab.ipynb>
2. Secrets (key icon, left sidebar) → add `HF_TOKEN` = your token → turn on
   **Notebook access**.
3. **Runtime → Change runtime type → T4 GPU**, then **Runtime → Run all**.
   The first start downloads about 8 GB, which takes several minutes.
4. The last cell prints `Running on public URL: https://….gradio.live`. That
   URL is your `HF_SPACE_ID` in step 5. Open it to test the model in its chat
   box, in English and in Nepali.

Limits: keep the Colab tab open. Free Colab stops after about 90 minutes
without interaction and after 12 hours at most, and **the link changes on
every restart**, so update `HF_SPACE_ID` on Render each time (Render
restarts the app, which takes a minute). On demo day, start Colab about 45
minutes early.

### Option B (paid, always on): Hugging Face GPU Space

Needs a payment method on your Hugging Face account. Billed per hour while
the Space is awake.

1. New Space → name it (e.g. `swasthyasetu-medgemma`) → SDK **Gradio** →
   hardware **A10G small** or **L4** → visibility **Private**.
2. Upload `app.py`, `requirements.txt` and `README.md` from
   [`model-space/`](../model-space/) (Files → Upload). `README.md` carries
   the Space config.
3. Space Settings → Variables and secrets → **New secret**: `HF_TOKEN` = your
   token.
4. Wait for it to build (first start downloads about 8 GB), then test it in
   the chat box on the Space page.
5. Settings → **Sleep time**, so you don't pay while it's idle. A sleeping
   Space takes a few minutes to wake, so wake it before a demo.

Your `HF_SPACE_ID` is `your-username/swasthyasetu-medgemma`. With a PRO
account you can pick **ZeroGPU** instead; `app.py` supports it.

## 2. TURN relay

Without TURN, calls only connect when both laptops can reach each other more
or less directly. That often fails across venue Wi-Fi, hotspots, and mobile
networks. A 5-minute relayed video call uses roughly 100–200 MB.

### Option A (free, no card): ExpressTURN

1. Sign up at <https://www.expressturn.com/> (free plan: 1,000 GB a month,
   one server location).
2. In the dashboard, copy the **server address**, **username**, and
   **password**.
3. In step 5 set (replace `HOST:PORT` with the address shown):

   | Variable | Value |
   |---|---|
   | `TURN_URLS` | `turn:HOST:PORT,turn:HOST:PORT?transport=tcp` |
   | `TURN_USERNAME` | the username |
   | `TURN_CREDENTIAL` | the password |

   The `?transport=tcp` entry gets through networks that block UDP. If the
   dashboard also lists port 443, add `turn:HOST:443?transport=tcp` too.

### Option B: Cloudflare Realtime TURN

The first 1,000 GB a month are free, then $0.05/GB, but Cloudflare asks for a
card on file. Realtime → TURN Server → create a key, then copy the **Turn
Token ID** and **API Token** into `CLOUDFLARE_TURN_KEY_ID` and
`CLOUDFLARE_TURN_API_TOKEN`.

Fill in one option; leave the other's variables empty.

## 3. Database (Neon, free Postgres)

Accounts, professional applications (with their document photos), review
decisions and saved chats live here. Without it the app falls back to a
SQLite file that Render wipes on every restart.

1. Sign up at <https://neon.tech> and create a project. Pick the region
   closest to your Render service's region (shown on the Render service
   page).
2. On the project dashboard, **Connect** → copy the connection string. It
   looks like `postgresql://user:password@ep-xxxx.region.aws.neon.tech/neondb?sslmode=require`.
3. That is your `DATABASE_URL` in step 5. Tables are created automatically on
   the first start.

*Alternative:* Supabase Postgres. Use its **Session pooler** connection
string (the direct one is IPv6-only on the free plan, which Render can't
reach). Free Supabase projects pause after a week without activity.

## 4. Google sign-in (OAuth client)

1. Open <https://console.cloud.google.com>, create a project (e.g.
   "SwasthyaSetu").
2. **Google Auth Platform** (APIs & Services → OAuth consent screen) →
   **Get started**: app name *SwasthyaSetu*, your support email, audience
   **External**, your contact email → Create.
3. **Audience** → **Publish app**. The app only asks for name, email and
   profile picture (non-sensitive scopes), so Google's app verification is
   not required. (While it is in "Testing", only test users you list can
   sign in.)
4. **Clients** → **Create client** → type **Web application**:
   - Authorized JavaScript origins: `https://swasthyasetu-xxxx.onrender.com`
   - Authorized redirect URIs:
     `https://swasthyasetu-xxxx.onrender.com/api/v1/auth/google/callback`
     (exactly this, with your Render URL; add
     `http://localhost:8000/api/v1/auth/google/callback` too for local runs)
5. Copy the **Client ID** and **Client secret**: `GOOGLE_CLIENT_ID` and
   `GOOGLE_CLIENT_SECRET` in step 5. Keep the secret out of chat and git.

## 5. The app (Render)

1. Render dashboard → **New → Blueprint** → connect this GitHub repo. It reads
   [`render.yaml`](../render.yaml).
2. Fill in the secrets it asks for. **Already deployed?** Render only asks
   when a service is created: add any missing ones in the service's
   **Environment** tab instead, then save (it redeploys).

   | Variable | Value |
   |---|---|
   | `HF_SPACE_ID` | Colab: the `https://….gradio.live` URL. Space: `your-username/swasthyasetu-medgemma` |
   | `HF_TOKEN` | Colab: leave empty. Space: the read token from step 1 |
   | `TURN_URLS`, `TURN_USERNAME`, `TURN_CREDENTIAL` | ExpressTURN values from step 2 |
   | `CLOUDFLARE_TURN_KEY_ID`, `CLOUDFLARE_TURN_API_TOKEN` | leave empty (unless you chose Cloudflare) |
   | `DATABASE_URL` | the Neon connection string from step 3 |
   | `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` | from step 4 |
   | `ADMIN_EMAILS` | the Google account email of the admin who reviews applications (comma-separate several) |

   Paste values with no spaces before or after them.
3. Deploy. The Docker build takes a few minutes. You get a URL like
   `https://swasthyasetu-xxxx.onrender.com`. Google's redirect address uses
   it automatically (from Render's `RENDER_EXTERNAL_URL`).
4. Open `https://…onrender.com/health`. You should see:
   - `"model_configured": true` and `"turn_configured": true`
   - `"database": "postgres"` and `"database_ready": true`
   - `"google_sign_in_configured": true`

## 6. First admin and first verified professional

1. On the site, **Sign in** with the admin's Google account. The account
   page shows **Admin: review applications**.
2. On another browser (or a private window), sign in with the professional's
   Google account → choose **Doctor** (or Pharmacist / MBBS Student) → fill
   in the verification form and upload the documents → **Submit for
   review**.
3. As the admin, open `/admin/` → check the documents and look the NMC /
   Pharmacy Council number up on the council's register → **Approve**.
4. The professional opens `/doctor/` → **Check again** → **Go online**.

For the demo, have the demo doctor approved beforehand.

**Plan:** `render.yaml` uses the free plan. It sleeps after 15 minutes idle;
the next visit takes about a minute to wake, and sleeping drops any open call.
For demo day, switch the service to **Starter** in Render (Settings →
Instance Type), or at least open the URL a few minutes beforehand.

Pushes to `main` redeploy automatically.

## 7. Verify before every rehearsal

The deployed site only has Google sign-in, so the smoke test borrows the
session of your already verified demo professional:

1. In Chrome, signed in on the site as the verified professional, press
   **F12** → **Application** → **Cookies** → your site → copy the value of
   `swasthya_session`. Treat it like a password.
2. Run:

```bash
pip install playwright
python scripts/smoke_test.py https://swasthyasetu-xxxx.onrender.com --pro-session <that value>
```

It runs the demo in Chrome windows: the professional goes online, chat,
emergency banner, doctor request (no login), live two-way video, the
"Verified Doctor" badge, mute, refresh-rejoin, and hang-up. It prints
`ALL GOOD — ready to demo.` or says which step failed. It uses your
installed Chrome with a fake camera.

Locally (see below) it also covers applying with documents, admin approval,
and a patient's saved chats.

Then do it once for real, on **two laptops on different networks**: one on
Wi-Fi and one on a phone hotspot. This is the only test that proves the TURN
relay works. Both run on one machine in the smoke test, so it can't.

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| Chat says "not configured" | `HF_SPACE_ID` not set on Render |
| Chat says "unavailable" / "took too long" | Colab stopped or its link changed (update `HF_SPACE_ID`); Space asleep (wait and retry), building, or `HF_TOKEN` can't access it |
| Space build fails on the model download | Model terms not accepted, or `HF_TOKEN` secret missing on the Space |
| Call stuck on "Connecting…", then "Connection problem" | TURN not configured, or wrong keys. Check `/health` → `turn_configured` |
| "Camera access needs a secure (https) connection" | Opened over plain http on a LAN address. Use the Render https URL |
| Everything slow for the first minute | Render free plan waking up |
| Google says "Error 400: redirect_uri_mismatch" | The redirect URI in the Google client must be exactly `https://<your Render URL>/api/v1/auth/google/callback` |
| Google says "Access blocked" / only some accounts can sign in | The app is still in "Testing": publish it (step 4.3) or add the accounts as test users |
| `/health` shows `"database": "sqlite"` | `DATABASE_URL` isn't set on Render: accounts and approvals would vanish on restart |
| `/health` shows `"database_ready": false` | `DATABASE_URL` is wrong or Neon is unreachable. Chat and patient calls still work; sign-in doesn't |
| The admin doesn't see "Admin: review applications" | `ADMIN_EMAILS` doesn't match the Google account's email exactly |
| A professional sees "Waiting for review" after approval | They need to press **Check again** (or reload) |

## Local development

```bash
# Backend: API on :8000
cd backend
python -m venv .venv && .venv/Scripts/pip install -r requirements-dev.txt   # Windows
cp .env.example .env            # set HF_SPACE_ID / HF_TOKEN; DEV_LOGIN=true
.venv/Scripts/uvicorn app.main:app --reload

# Frontend: on :3000, calling the API on :8000
cd frontend
cp .env.example .env.local
npm install && npm run dev
```

Camera access works on `localhost`, so you can test a call with two browser
windows on one machine. Use headphones to avoid feedback. Some laptops only
let one window use the camera; the second then falls back to voice-only.

Locally the database is a SQLite file and `DEV_LOGIN=true` adds a
"development sign-in" form (any email, no Google), so you can test as a
patient, a professional and the admin (an email listed in `ADMIN_EMAILS`).
It is always off in production.

The full local smoke test (with admin@smoke.test as admin):

```bash
cd frontend && npm run build && cd ../backend
STATIC_DIR=../frontend/out DEV_LOGIN=true ADMIN_EMAILS=admin@smoke.test \
  .venv/Scripts/uvicorn app.main:app --port 8000
python ../scripts/smoke_test.py http://localhost:8000
```

To run exactly what production runs: `docker compose up --build`, then
open <http://localhost:8000>.
