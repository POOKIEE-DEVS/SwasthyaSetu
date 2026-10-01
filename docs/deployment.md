# Deploying SwasthyaSetu

Three pieces, each on a platform suited to it:

| Piece | Where | Why there |
|---|---|---|
| MedGemma model | Google Colab GPU (free) or a Hugging Face GPU Space (paid) | Needs a GPU. Loads the gated model with your token. |
| App (website + API + call signalling) | Render, one Docker web service | HTTPS (required for cameras) and WebSockets, from one origin. |
| TURN relay for video | Cloudflare Realtime TURN | Calls between different networks need a relay. Render has no UDP. |

Do them in this order. Allow about an hour the first time.

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
   URL is your `HF_SPACE_ID` in step 3. Open it to test the model in its chat
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
3. In step 3 set (replace `HOST:PORT` with the address shown):

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

## 3. The app (Render)

1. Render dashboard → **New → Blueprint** → connect this GitHub repo. It reads
   [`render.yaml`](../render.yaml).
2. Fill in the secrets it asks for:

   | Variable | Value |
   |---|---|
   | `HF_SPACE_ID` | Colab: the `https://….gradio.live` URL. Space: `your-username/swasthyasetu-medgemma` |
   | `HF_TOKEN` | Colab: leave empty. Space: the read token from step 1 |
   | `TURN_URLS`, `TURN_USERNAME`, `TURN_CREDENTIAL` | ExpressTURN values from step 2 |
   | `CLOUDFLARE_TURN_KEY_ID`, `CLOUDFLARE_TURN_API_TOKEN` | leave empty (unless you chose Cloudflare) |

3. Deploy. The Docker build takes a few minutes. You get a URL like
   `https://swasthyasetu-xxxx.onrender.com`.
4. Open `https://…onrender.com/health`. You should see
   `"model_configured": true` and `"turn_configured": true`.

**Plan:** `render.yaml` uses the free plan. It sleeps after 15 minutes idle;
the next visit takes about a minute to wake, and sleeping drops any open call.
For demo day, switch the service to **Starter** in Render (Settings →
Instance Type), or at least open the URL a few minutes beforehand.

Pushes to `main` redeploy automatically.

## 4. Verify before every rehearsal

```bash
pip install playwright
python scripts/smoke_test.py https://swasthyasetu-xxxx.onrender.com
```

The smoke test runs the whole demo in two Chrome windows: chat, emergency
banner, doctor request, live two-way video, mute, refresh-rejoin, and hang-up.
It prints `ALL GOOD — ready to demo.` or says which step failed. It uses your
installed Chrome with a fake camera.

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

## Local development

```bash
# Backend: API on :8000
cd backend
python -m venv .venv && .venv/Scripts/pip install -r requirements-dev.txt   # Windows
cp .env.example .env            # set HF_SPACE_ID / HF_TOKEN
.venv/Scripts/uvicorn app.main:app --reload

# Frontend: on :3000, calling the API on :8000
cd frontend
cp .env.example .env.local
npm install && npm run dev
```

Camera access works on `localhost`, so you can test a call with two browser
windows on one machine. Use headphones to avoid feedback. Some laptops only
let one window use the camera; the second then falls back to voice-only.

To run exactly what production runs: `docker compose up --build`, then
open <http://localhost:8000>.
