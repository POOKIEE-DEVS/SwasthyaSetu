"""Pre-demo smoke test: the whole demo flow in two real Chrome windows.

Runs the patient and the doctor side by side, with Chrome's fake camera and
microphone:

  doctor goes online -> patient chats (English, then urgent Nepali)
  -> patient requests a doctor -> doctor sees them live, with the shared chat
  -> doctor accepts -> two-way audio+video connects
  -> mute / camera off -> patient refreshes mid-call and the call re-establishes
  -> doctor hangs up -> both sides end cleanly

Run it against the deployed URL before every rehearsal and the demo itself:

    pip install playwright          # uses your installed Chrome
    python scripts/smoke_test.py https://your-app.onrender.com

Add --shots DIR to save screenshots. Exits non-zero if any step fails.
Against the real model the first reply can take a while (Space cold start);
the chat step waits up to 3 minutes.
"""

from __future__ import annotations

import argparse
import sys
import time
from pathlib import Path

from playwright.sync_api import expect, sync_playwright

REMOTE_PLAYING = """() => {
  const v = document.querySelectorAll('video')[0];
  return !!v && !!v.srcObject && v.videoWidth > 0 && v.readyState >= 2;
}"""
REMOTE_TRACKS = """() => {
  const s = document.querySelectorAll('video')[0].srcObject;
  if (!s) return null;
  return {audio: s.getAudioTracks().length, video: s.getVideoTracks().length};
}"""
STREAM_ID = "() => document.querySelectorAll('video')[0]?.srcObject?.id ?? null"

# Records every RTCPeerConnection the page creates, so a failed call can be
# diagnosed from outside without shipping debug code in the app.
PC_SPY = """(() => {
  const Original = window.RTCPeerConnection;
  if (!Original) return;
  window.__pcs = [];
  window.RTCPeerConnection = function (...args) {
    const pc = new Original(...args);
    window.__pcs.push(pc);
    return pc;
  };
  window.RTCPeerConnection.prototype = Original.prototype;
})();"""

# Connection, track, and RTP counters for every peer connection and video.
CALL_DIAGNOSTICS = """async () => {
  const pcs = [];
  for (const pc of window.__pcs ?? []) {
    const rtp = [];
    try {
      (await pc.getStats()).forEach((r) => {
        if (r.type === "inbound-rtp" || r.type === "outbound-rtp") {
          rtp.push(`${r.type}/${r.kind} packets=${r.packetsReceived ?? r.packetsSent}`
            + ` frames=${r.framesDecoded ?? r.framesEncoded ?? "-"}`);
        }
      });
    } catch (e) { rtp.push(`getStats failed: ${e}`); }
    pcs.push({
      conn: pc.connectionState, ice: pc.iceConnectionState, sig: pc.signalingState,
      receivers: pc.getReceivers().map((r) => `${r.track.kind}:${r.track.readyState}`
        + `${r.track.muted ? ":muted" : ""}`),
      rtp,
    });
  }
  const videos = [...document.querySelectorAll("video")].map((v) => ({
    readyState: v.readyState, size: `${v.videoWidth}x${v.videoHeight}`,
    paused: v.paused, muted: v.muted,
    tracks: v.srcObject ? v.srcObject.getTracks().map((t) =>
      `${t.kind}:${t.readyState}${t.enabled ? "" : ":disabled"}`
      + `${t.muted ? ":muted" : ""}`) : null,
  }));
  return {pcs, videos};
}"""


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("base_url", help="e.g. https://your-app.onrender.com")
    parser.add_argument("--shots", type=Path, help="save screenshots here")
    parser.add_argument("--headed", action="store_true", help="show the browsers")
    args = parser.parse_args()
    base = args.base_url.rstrip("/")
    if args.shots:
        args.shots.mkdir(parents=True, exist_ok=True)

    results: list[tuple[str, bool, str]] = []
    errors: dict[str, list[str]] = {"doctor": [], "patient": []}
    # Recent console output of every level, for diagnosing a failed step.
    logs: dict[str, list[str]] = {"doctor": [], "patient": []}

    def shot(page, name: str) -> None:
        if args.shots:
            page.screenshot(path=str(args.shots / f"{name}.png"))

    def step(name: str, fn) -> None:
        started = time.time()
        try:
            detail = fn() or ""
        except Exception as exc:
            results.append((name, False, f"{type(exc).__name__}: {str(exc)[:300]}"))
            raise
        results.append((name, True, f"{detail} ({time.time() - started:.1f}s)".strip()))

    with sync_playwright() as pw:
        browser = pw.chromium.launch(
            channel="chrome",
            headless=not args.headed,
            args=[
                "--use-fake-device-for-media-stream",
                "--use-fake-ui-for-media-stream",
                "--autoplay-policy=no-user-gesture-required",
            ],
        )

        def new_page(role: str):
            context = browser.new_context(
                permissions=["camera", "microphone"],
                viewport={"width": 1280, "height": 800},
            )
            context.add_init_script(PC_SPY)
            page = context.new_page()

            def on_console(m, role=role):
                logs[role] = [*logs[role][-19:], f"[{m.type}] {m.text}"]
                if m.type == "error":
                    errors[role].append(m.text)

            page.on("console", on_console)
            page.on("pageerror", lambda e: errors[role].append(f"pageerror: {e}"))
            return page

        doctor = new_page("doctor")
        patient = new_page("patient")

        def doctor_online():
            doctor.goto(f"{base}/doctor/")
            doctor.get_by_role("button", name="Go online").click()
            expect(doctor.get_by_text("Online · you'll hear a tone")).to_be_visible(
                timeout=15000
            )

        def patient_chats():
            patient.goto(f"{base}/patient/")
            box = patient.get_by_label("Message")
            box.fill("I burned my hand on the stove")
            box.press("Enter")
            reply = patient.locator('[data-role="assistant"]').first
            expect(reply).to_be_visible(timeout=180_000)
            text = reply.inner_text()
            return f"reply: {text[:60]!r}"

        def urgent_nepali():
            box = patient.get_by_label("Message")
            box.fill("मेरो बुबाको छाती दुख्यो")
            box.press("Enter")
            expect(patient.get_by_text("This may be an emergency")).to_be_visible(
                timeout=180_000
            )
            shot(patient, "1-patient-chat")

        def request_doctor():
            patient.get_by_role(
                "button", name="Talk to a doctor · डाक्टरसँग कुरा गर्नुहोस्"
            ).click()
            patient.get_by_label("Your name · तपाईंको नाम").fill("Smoke Test")
            patient.get_by_role("button", name="Request a doctor").click()
            expect(patient.get_by_text("Waiting for a doctor to accept")).to_be_visible(
                timeout=20000
            )

        def doctor_sees_patient():
            card = doctor.get_by_role("listitem").filter(has_text="Smoke Test")
            expect(card).to_be_visible(timeout=15000)
            card.get_by_text("Read their chat with the assistant").click()
            expect(
                card.get_by_text("I burned my hand on the stove").first
            ).to_be_visible()
            shot(doctor, "2-doctor-queue")

        def call_connects():
            card = doctor.get_by_role("listitem").filter(has_text="Smoke Test")
            card.get_by_role("button", name="Accept").click()
            doctor.wait_for_function(REMOTE_PLAYING, timeout=45000)
            patient.wait_for_function(REMOTE_PLAYING, timeout=45000)
            d, p = doctor.evaluate(REMOTE_TRACKS), patient.evaluate(REMOTE_TRACKS)
            assert d == {"audio": 1, "video": 1}, f"doctor received {d}"
            assert p == {"audio": 1, "video": 1}, f"patient received {p}"
            time.sleep(1.5)
            shot(doctor, "3-doctor-call")
            shot(patient, "4-patient-call")
            return f"doctor recv {d}, patient recv {p}"

        def controls():
            patient.get_by_role("button", name="Mute microphone").click()
            expect(
                patient.get_by_role("button", name="Unmute microphone")
            ).to_be_visible()
            muted = patient.evaluate(
                "() => document.querySelectorAll('video')[1].srcObject"
                ".getAudioTracks()[0].enabled === false"
            )
            assert muted, "mic track still enabled after Mute"
            patient.get_by_role("button", name="Turn camera off").click()
            expect(patient.get_by_text("Camera off").first).to_be_visible()

        def refresh_rejoins():
            before = doctor.evaluate(STREAM_ID)
            patient.reload()
            patient.wait_for_function(REMOTE_PLAYING, timeout=45000)
            doctor.wait_for_function(
                "() => { const v = document.querySelectorAll('video')[0];"
                " return !!v && !!v.srcObject && v.srcObject.id !== "
                + repr(before)
                + " && v.videoWidth > 0 && v.readyState >= 2; }",
                timeout=45000,
            )

        def hang_up():
            doctor.get_by_role("button", name="End call").click()
            expect(patient.get_by_text("The doctor ended the call.")).to_be_visible(
                timeout=15000
            )
            doctor.get_by_role("button", name="Done").click()
            patient.get_by_role("button", name="Done").click()
            expect(
                patient.get_by_text("I burned my hand on the stove").first
            ).to_be_visible()

        steps = [
            ("doctor goes online", doctor_online),
            ("patient chat gets a model reply", patient_chats),
            ("urgent Nepali message shows emergency banner", urgent_nepali),
            ("patient requests a doctor", request_doctor),
            ("doctor queue shows patient + shared chat", doctor_sees_patient),
            ("video call connects both ways (audio + video)", call_connects),
            ("mute and camera-off toggle the real tracks", controls),
            ("patient refresh mid-call re-establishes the call", refresh_rejoins),
            ("hang up ends both sides, chat kept", hang_up),
        ]
        try:
            for name, fn in steps:
                step(name, fn)
        except Exception:
            # Show what each side was looking at when the step failed.
            for role, page in (("doctor", doctor), ("patient", patient)):
                try:
                    screen = page.evaluate(
                        "() => document.querySelector('main')?.innerText ?? ''"
                    )
                except Exception as exc:
                    screen = f"(unavailable: {exc})"
                print(f"\n--- {role} screen at failure ---\n{screen[:300]}")
                try:
                    diag = page.evaluate(CALL_DIAGNOSTICS)
                    for i, pc in enumerate(diag["pcs"]):
                        print(f"    {role} peer#{i}: {pc}")
                    for i, video in enumerate(diag["videos"]):
                        print(f"    {role} video#{i}: {video}")
                except Exception as exc:
                    print(f"    {role} diagnostics unavailable: {exc}")
                for line in logs[role][-10:]:
                    print(f"    {role} console: {line[:220]}")
        finally:
            browser.close()

    print()
    for name, ok, detail in results:
        print(f"{'PASS' if ok else 'FAIL'}  {name}  {detail}")
    skipped = len(steps) - len(results)
    if skipped:
        print(f"SKIP  {skipped} later step(s) not run")
    for role, errs in errors.items():
        if errs:
            print(f"\nbrowser console errors ({role}):")
            for e in errs[:8]:
                print("   ", e[:200])

    ok = len(results) == len(steps) and all(r[1] for r in results)
    print("\nALL GOOD — ready to demo." if ok else "\nNOT READY — see failures above.")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
