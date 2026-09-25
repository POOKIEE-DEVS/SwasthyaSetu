# Demo playbook

A 5-minute live demo with two laptops: **Patient** (presenter) and **Doctor**
(teammate). Both use the deployed https URL.

## Script

| Time | Who | What to do and say |
|---|---|---|
| 0:00–0:40 | Presenter | **The problem.** In rural Nepal the nearest doctor can be hours away. In an emergency, people need guidance now and a real doctor soon. |
| 0:40–1:50 | Patient laptop | Open `/patient/`. Type in English: *"I burned my hand while cooking."* MedGemma replies with first-aid steps. Ask a follow-up (*"Should I put ice on it?"*) to show it remembers the conversation. |
| 1:50–2:30 | Patient laptop | Switch to Nepali: *"मेरो बुबाको छाती दुख्यो"* ("my father has chest pain"). The red emergency banner appears straight away, with **Call 102** and **Talk to a doctor**. Point out that it doesn't wait for the AI. |
| 2:30–3:00 | Both | Patient: **Talk to a doctor** → name → leave "share my chat" ticked → **Request a doctor**. Doctor laptop, already online, beeps and shows the patient **with the chat they shared**. Doctor: **Accept**. |
| 3:00–4:10 | Both | Live video call. Show **Mute** and **Camera off**. The doctor reads the shared chat on the side panel, so the patient doesn't have to repeat themselves. Doctor: **End call**. |
| 4:10–5:00 | Presenter | **How it works:** Next.js PWA + FastAPI; MedGemma on a Hugging Face GPU; WebRTC peer-to-peer with a TURN relay. **Honest limits:** first-aid information, not diagnosis; no login yet. **Next:** accounts, doctor verification, offline first-aid library. |

## Checklist

**The day before**
- [ ] `python scripts/smoke_test.py <URL>` prints `ALL GOOD`
- [ ] One real call between two laptops on **different networks** (Wi-Fi + phone hotspot)
- [ ] Try the exact demo sentences on the real model and check the replies are sensible, in both languages
- [ ] Record a backup video of the full flow

**30 minutes before**
- [ ] Wake the Space: open it and send one message; wait until it replies
- [ ] Open the app URL on both laptops (wakes Render if on the free plan)
- [ ] Run the smoke test once more
- [ ] Doctor laptop: open `/doctor/`, **Go online**, volume up (for the arrival tone)
- [ ] Patient laptop: open `/patient/`, press **New chat** so the history is clean
- [ ] Allow camera and mic on both. Plug in chargers. Phone hotspot ready.

## If something goes wrong

| Problem on stage | What to do |
|---|---|
| AI reply is slow | Say the model runs on a GPU in the cloud, and move on. The retry button is there if it errors. |
| AI is down | Skip to the doctor call; it doesn't depend on the AI. Show the backup video's chat part. |
| Call stuck on "Connecting…" | Give it 20 seconds: it retries by itself. If **Reconnect** appears, press it. Next, move both laptops to the phone hotspot. |
| Venue Wi-Fi dead | Both laptops on the phone hotspot. |
| Everything fails | Play the backup video, and talk over it. |

## Likely judge questions

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
