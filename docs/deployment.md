# Deploying SwasthyaSetu

Three pieces, each on a platform suited to it:

| Piece | Where | Why there |
|---|---|---|
| MedGemma model | Hugging Face Space (GPU) | Needs a GPU. The Space loads the gated model with your token. |
| App (website + API + call signalling) | Render, one Docker web service | HTTPS (required for cameras) and WebSockets, from one origin. |
| TURN relay for video | Cloudflare Realtime TURN | Calls between different networks need a relay. Render has no UDP. |

Do them in this order. Allow about an hour the first time.

---

## 1. MedGemma Space

1. **Accept the model terms.** Signed in to Hugging Face, open
   [google/medgemma-1.5-4b-it](https://huggingface.co/google/medgemma-1.5-4b-it)
   and accept the Health AI Developer Foundations terms. The Space cannot
   download the model until your account has done this.
2. **Create a token.** Settings → Access Tokens → create a **read** token.
   Keep it private: it goes into two dashboards below and nowhere else (not
   into chat, not into git).
3. **Create the Space.** New Space → name it (e.g. `swasthyasetu-medgemma`) →
   SDK **Gradio** → hardware **A10G small** or **L4** → visibility **Private**.
4. **Add the code.** Upload the three files from [`model-space/`](../model-space/)
   (`app.py`, `requirements.txt`, `README.md`) via Files → Upload, or push
   them to the Space's git repo. `README.md` carries the Space config.
5. **Add the secret.** Space Settings → Variables and secrets → **New secret**:
   `HF_TOKEN` = your token.
6. **Wait for it to build.** The first start downloads about 8 GB of model
   weights, which takes several minutes. Then test it in the chat box on the
   Space page, in English and in Nepali.
7. **Set a sleep time.** GPU hardware bills per hour while the Space is
   running. Settings → Sleep time. A sleeping Space wakes on the next request,
   which takes a few minutes, so wake it before a demo.

> **Cheaper alternative:** with a Hugging Face PRO account you can pick
> **ZeroGPU** hardware instead. `app.py` already supports it (`@spaces.GPU`).
> GPU time then comes from a daily quota instead of an hourly bill.

Your Space ID is `your-username/swasthyasetu-medgemma`. You need it in step 3.

## 2. TURN relay (Cloudflare)

Without TURN, calls only connect when both laptops can reach each other more
or less directly. That often fails across venue Wi-Fi, hotspots, and mobile
networks.

1. In the Cloudflare dashboard, open **Realtime → TURN Server** and create a
   TURN key.
2. Copy the **Turn Token ID** and the **API Token**.

The first 1,000 GB each month are free (shared with Cloudflare's SFU), then
$0.05/GB. A 5-minute call uses well under 1 GB.

*Alternative:* any TURN provider with static credentials, for example Metered.
Set `TURN_URLS`, `TURN_USERNAME`, `TURN_CREDENTIAL` instead of the Cloudflare
variables.

## 3. The app (Render)

1. Render dashboard → **New → Blueprint** → connect this GitHub repo. It reads
   [`render.yaml`](../render.yaml).
2. Fill in the secrets it asks for:

   | Variable | Value |
   |---|---|
   | `HF_SPACE_ID` | `your-username/swasthyasetu-medgemma` |
   | `HF_TOKEN` | the read token from step 1 |
   | `CLOUDFLARE_TURN_KEY_ID` | Turn Token ID |
   | `CLOUDFLARE_TURN_API_TOKEN` | API Token |

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
| Chat says "unavailable" / "took too long" | Space asleep (wait and retry), building, or `HF_TOKEN` can't access it |
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
