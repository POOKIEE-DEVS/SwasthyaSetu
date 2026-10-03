# Demo playbook

A 5-minute live demo with two laptops: **Patient** (presenter) and **Doctor**
(teammate, signed in with a Google account that the admin has already
**verified**). Both use the deployed https URL. A third tab on the presenter
laptop is signed in as the **admin**, for the trust beat.

## Script

| Time | Who | What to do and say |
|---|---|---|
| 0:00–0:30 | Presenter | **The problem.** In rural Nepal the nearest doctor can be hours away. In an emergency, people need guidance now and a real doctor soon. |
| 0:30–1:30 | Patient laptop | Open the site: the hero puts **the live first-aid chat** beside "First aid now. A verified doctor next.", with 102 always visible. Optionally tap **नेपाली** in the header to show the page in Nepali. No sign-up. Type in English: *"I burned my hand while cooking."* MedGemma replies with first-aid steps. Ask a follow-up (*"Should I put ice on it?"*) to show it remembers the conversation. |
| 1:30–2:00 | Patient laptop | Switch to Nepali: *"मेरो बुबाको छाती दुख्यो"* ("my father has chest pain"). The red emergency banner appears straight away, with **Call 102** and **Talk to a professional**. Point out that it doesn't wait for the AI. |
| 2:00–2:30 | Both | Patient: **Talk to a professional** → name → leave "share my chat" ticked → **Request a doctor** (still no login). Doctor laptop, online under the green **Verified Doctor** banner, beeps and shows the patient **with the chat they shared**. Doctor: **Accept**. |
| 2:30–3:30 | Both | Live video call. Point at **"Verified Doctor · Dr. …"** on the patient's screen. Show **Mute** and **Camera off**. The doctor reads the shared chat on the side panel, so the patient doesn't have to repeat themselves. Doctor: **End call**. |
| 3:30–4:15 | Presenter | **Who is allowed to answer?** Admin tab → `/admin/` → **Approved**: the doctor's card with NMC number, citizenship and certificate. Then **Waiting for review**: a pharmacist or MBBS student application (student: doctor's letter of recommendation). Only approved professionals ever see a patient. Optional: patient taps **Sign in to save chats** → the conversation appears under **My chats**. |
| 4:15–5:00 | Presenter | **How it works:** Next.js PWA + FastAPI + Postgres; Google sign-in; MedGemma on a cloud GPU; WebRTC peer-to-peer with a TURN relay. **Honest limits:** first-aid information, not diagnosis; verification is a manual document check. **Next:** link to the council registers, offline first-aid library, Nepali voice. |

## Checklist

**The day before**
- [ ] The demo doctor's Google account has applied and the admin has **approved** it
- [ ] One more application (pharmacist or MBBS student) is left **pending**, to show on the admin page
- [ ] `python scripts/smoke_test.py <URL> --pro-session <cookie>` prints `ALL GOOD` (see deployment.md §7)
- [ ] One real call between two laptops on **different networks** (Wi-Fi + phone hotspot)
- [ ] Try the exact demo sentences on the real model and check the replies are sensible, in both languages
- [ ] Record a backup video of the full flow

**30 minutes before**
- [ ] Start the model (Colab: Run all, ~45 minutes before; put the new link in Render's `HF_SPACE_ID`). Send it one message and wait for the reply
- [ ] Open the app URL on both laptops (wakes Render if on the free plan)
- [ ] Run the smoke test once more
- [ ] Doctor laptop: signed in as the verified doctor, open `/doctor/`, **Go online**, volume up (for the arrival tone)
- [ ] Patient laptop: open the site **signed out**, press **New chat** so the conversation is clean
- [ ] Patient laptop, second tab: signed in as the admin on `/admin/`
- [ ] Allow camera and mic on both. Plug in chargers. Phone hotspot ready.

## If something goes wrong

| Problem on stage | What to do |
|---|---|
| AI reply is slow | Say the model runs on a GPU in the cloud, and move on. The retry button is there if it errors. |
| AI is down | Skip to the doctor call; it doesn't depend on the AI. Show the backup video's chat part. |
| Call stuck on "Connecting…" | Give it 20 seconds: it retries by itself. If **Reconnect** appears, press it. Next, move both laptops to the phone hotspot. |
| Venue Wi-Fi dead | Both laptops on the phone hotspot. |
| Doctor page says "Waiting for review" | Admin tab → approve; doctor presses **Check again**. |
| Google sign-in fails on stage | Skip it: the patient journey needs no login, and the doctor laptop is already signed in. |
| Everything fails | Play the backup video, and talk over it. |

## Likely judge questions

- **"Can anyone sign up as a doctor?"** No. Every doctor, pharmacist and
  MBBS student uploads their citizenship certificate and their council
  registration (NMC or Nepal Pharmacy Council), or for students a doctor's
  letter of recommendation. An admin checks each one by hand against the
  council register. Until approved they can't see patients, and the server
  enforces it on every request, not just the UI.
- **"Why should a patient trust who answers?"** The call shows the
  professional's verified role and name: "Verified Doctor", "Verified
  Pharmacist" or "Verified MBBS Student".
- **"Does a patient need an account?"** Never for help. Signing in only
  saves their chats. In an emergency, nothing asks them to log in.
- **"What about the documents?"** Stored only to verify the applicant,
  visible only to the admin, and file types are checked so nothing
  executable can be uploaded.

- **"Is it diagnosing people?"** No. It gives first-aid information and
  always points to 102 and a real doctor for anything serious. The emergency
  banner is keyword-based on purpose, so it never waits for the model.
- **"Why MedGemma?"** It's a medically-tuned open model. We host it ourselves,
  so patient text doesn't go to a third-party chatbot API.
- **"Is the video call private?"** The media is encrypted end to end
  (DTLS-SRTP). The server only helps the two browsers find each other. When a
  relay is needed, it passes along encrypted media it can't read.
- **"What about bad connectivity?"** It falls back to voice-only when there's
  no camera. It shows clearly when you're offline, rejoins after a refresh,
  and keeps the emergency number on screen at all times.
