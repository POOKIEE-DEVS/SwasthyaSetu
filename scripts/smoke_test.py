"""Pre-demo smoke test: the whole demo flow in real Chrome windows.

Runs a professional, an admin and a patient side by side, with Chrome's
fake camera and microphone:

  professional signs in -> applies as a doctor with documents
  -> admin approves -> professional goes online
  -> patient chats (English, then urgent Nepali) -> requests a doctor
  -> professional sees them live, with the shared chat -> accepts
  -> two-way audio+video, patient sees "Verified Doctor" -> mute / camera off
  -> patient refreshes mid-call and the call re-establishes -> hang up
  -> the professional's help record counts the call, with the patient's name
  -> patient signs in and finds the conversation in "My chats"

Locally, run the server with DEV_LOGIN=true and ADMIN_EMAILS including
admin@smoke.test; the test then signs everyone in without Google:

    pip install playwright          # uses your installed Chrome
    python scripts/smoke_test.py http://localhost:8000

Against the deployed site (Google sign-in only), pass the session cookie of
an already verified professional; the sign-up, review and patient sign-in
steps are skipped:

    python scripts/smoke_test.py https://your-app.onrender.com \\
        --pro-session <value of the swasthya_session cookie>

Add --shots DIR to save screenshots. Exits non-zero if any step fails.
Against the real model the first reply can take a while (cold start); the
chat step waits up to 3 minutes.
"""

from __future__ import annotations

import argparse
import base64
import re
import sys
import tempfile
import time
import uuid
from pathlib import Path

from playwright.sync_api import expect, sync_playwright

ADMIN_EMAIL = "admin@smoke.test"
SESSION_COOKIE = "swasthya_session"
# A real 1x1 PNG, used for every uploaded document.
PNG = base64.b64decode(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
)


def write_fake_camera(path: Path) -> None:
    """A small Y4M clip (320x240, moving gradient) for Chrome to play as the
    camera. Chrome's built-in fake camera sometimes ends its own track a
    moment after it starts, or ~35 s in, with no page code involved (7 of 16
    tracks in a bare-page test on Windows); a file-backed camera doesn't."""
    width, height, frames = 320, 240, 20
    quarter = width * height // 4
    with path.open("wb") as f:
        f.write(f"YUV4MPEG2 W{width} H{height} F15:1 Ip A1:1 C420jpeg\n".encode())
        for n in range(frames):
            f.write(b"FRAME\n")
            luma = ((x + y + n * 8) % 256 for y in range(height) for x in range(width))
            f.write(bytes(luma))
            f.write(bytes([128]) * quarter)
            f.write(bytes([(n * 12) % 256]) * quarter)


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
})();
// Media events, for diagnosing a camera track that ends mid-call.
(() => {
  window.__media = [];
  const t0 = performance.now();
  const log = (what) =>
    window.__media.push(`${Math.round(performance.now() - t0)}ms ${what}`);
  const md = navigator.mediaDevices;
  if (!md) return;
  const gum = md.getUserMedia.bind(md);
  md.getUserMedia = async (c) => {
    log(`getUserMedia(${JSON.stringify(Object.keys(c || {}))})`);
    const stream = await gum(c);
    for (const t of stream.getTracks()) {
      log(`  got ${t.kind} ${t.id.slice(0, 6)}`);
      t.addEventListener("ended", () => log(`ended ${t.kind} ${t.id.slice(0, 6)}`));
    }
    return stream;
  };
  const stop = MediaStreamTrack.prototype.stop;
  MediaStreamTrack.prototype.stop = function () {
    // Who stopped it: our code calls stop(); the browser ending a track
    // shows up as "ended" with no stop() before it.
    const caller = (new Error().stack || "")
      .split(String.fromCharCode(10)).slice(2, 4).join(" | ").trim();
    log(`stop() ${this.kind} ${this.id.slice(0, 6)} <- ${caller}`);
    return stop.call(this);
  };
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
    parser.add_argument(
        "--pro-session",
        help="session cookie of a verified professional (for the deployed site)",
    )
    args = parser.parse_args()
    base = args.base_url.rstrip("/")
    if args.shots:
        args.shots.mkdir(parents=True, exist_ok=True)

    run_id = uuid.uuid4().hex[:6]
    pro_name = f"Dr. Smoke {run_id}"
    roles = ("doctor", "patient", "admin")
    results: list[tuple[str, bool, str]] = []
    errors: dict[str, list[str]] = {r: [] for r in roles}
    # Recent console output of every level, for diagnosing a failed step.
    logs: dict[str, list[str]] = {r: [] for r in roles}

    def shot(page, name: str) -> None:
        if args.shots:
            page.screenshot(path=str(args.shots / f"{name}.png"), full_page=True)

    def step(name: str, fn) -> None:
        started = time.time()
        try:
            detail = fn() or ""
        except Exception as exc:
            results.append((name, False, f"{type(exc).__name__}: {str(exc)[:300]}"))
            raise
        results.append((name, True, f"{detail} ({time.time() - started:.1f}s)".strip()))

    camera_file = Path(tempfile.gettempdir()) / "swasthyasetu-fake-camera.y4m"
    write_fake_camera(camera_file)

    with sync_playwright() as pw:

        def launch():
            return pw.chromium.launch(
                channel="chrome",
                headless=not args.headed,
                args=[
                    "--use-fake-device-for-media-stream",
                    "--use-fake-ui-for-media-stream",
                    "--autoplay-policy=no-user-gesture-required",
                    f"--use-file-for-fake-video-capture={camera_file}",
                ],
            )

        # One Chrome process per call participant, like two laptops.
        browsers = {"doctor": launch(), "patient": launch()}
        browsers["admin"] = browsers["doctor"]  # never uses the camera

        def new_page(role: str):
            context = browsers[role].new_context(
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
            page.on(
                "pageerror", lambda e, role=role: errors[role].append(f"pageerror: {e}")
            )
            return page

        doctor = new_page("doctor")
        patient = new_page("patient")
        admin = new_page("admin")

        me = doctor.request.get(f"{base}/api/v1/auth/me").json()
        dev_mode = bool(me.get("dev_login"))
        if not dev_mode and not args.pro_session:
            print(
                "This server has no development login. Pass --pro-session with the\n"
                "swasthya_session cookie of a verified professional (see --help)."
            )
            return 2

        def dev_sign_in(page, email: str, name: str) -> None:
            page.get_by_label("Development email").fill(email)
            page.get_by_label("Development name").fill(name)
            page.get_by_role("button", name="Sign in for testing").click()

        def professional_applies():
            doctor.goto(f"{base}/doctor/")
            expect(
                doctor.get_by_text("For medical professionals").first
            ).to_be_visible()
            doctor.get_by_role("link", name="Sign in", exact=True).last.click()
            dev_sign_in(doctor, f"doctor-{run_id}@smoke.test", pro_name)
            # Exact start: the student card mentions "a doctor" too.
            doctor.get_by_role("button", name=re.compile(r"^Doctor")).click()
            expect(doctor.get_by_role("heading", name="Get verified")).to_be_visible(
                timeout=15000
            )
            doctor.get_by_label("Full name, as on your citizenship").fill(pro_name)
            doctor.get_by_label("Phone number").fill("9841234567")
            doctor.get_by_label("Citizenship number").fill("27-01-71-12345")
            doctor.get_by_label("Issuing district").fill("Kathmandu")
            doctor.get_by_label("registration number").fill("12345")
            for label in (
                "Citizenship: front",
                "Citizenship: back",
                "NMC registration certificate",
            ):
                doctor.get_by_label(label).set_input_files(
                    {"name": "doc.png", "mimeType": "image/png", "buffer": PNG}
                )
            doctor.get_by_role("checkbox").check()
            shot(doctor, "0-apply-form")
            doctor.get_by_role("button", name="Submit for review").click()
            expect(doctor.get_by_text("Waiting for review")).to_be_visible(
                timeout=15000
            )

        def admin_approves():
            admin.goto(f"{base}/account/?next=/admin/")
            dev_sign_in(admin, ADMIN_EMAIL, "Smoke Admin")
            admin.wait_for_url("**/admin/", timeout=15000)
            card = admin.get_by_test_id("application").filter(has_text=pro_name)
            expect(card).to_be_visible(timeout=15000)
            # Every document is viewable by the admin.
            expect(card.get_by_role("img")).to_have_count(3)
            shot(admin, "1-admin-review")
            card.get_by_role("button", name="Approve").click()
            expect(card).to_have_count(0, timeout=15000)

        # The professional's "You've helped N people" before the call.
        helped_before: dict[str, int] = {}

        def professional_goes_online():
            if dev_mode:
                doctor.get_by_role("button", name="Check again").click()
            else:
                doctor.context.add_cookies(
                    [{"name": SESSION_COOKIE, "value": args.pro_session, "url": base}]
                )
                doctor.goto(f"{base}/doctor/")
            expect(doctor.get_by_text("Verified Doctor").first).to_be_visible(
                timeout=15000
            )
            doctor.get_by_role("button", name="Go online").click()
            help_count = doctor.get_by_test_id("help-count")
            expect(help_count).to_be_visible(timeout=15000)
            helped_before["count"] = int(help_count.inner_text())
            expect(doctor.get_by_text("Online · you'll hear a tone")).to_be_visible(
                timeout=15000
            )

        def patient_chats():
            # The landing page's main action opens the chat, no sign-in.
            patient.goto(f"{base}/")
            patient.get_by_role("link", name="Get first-aid help").first.click()
            patient.wait_for_url(f"{base}/patient/", timeout=15000)
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
            expect(patient.locator('[data-role="assistant"]')).to_have_count(
                2, timeout=180_000
            )
            shot(patient, "2-patient-chat")

        def request_doctor():
            # The chat's own button.
            patient.get_by_test_id("talk-to-professional").click()
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
            shot(doctor, "3-doctor-queue")

        def call_connects():
            card = doctor.get_by_role("listitem").filter(has_text="Smoke Test")
            card.get_by_role("button", name="Accept").click()
            doctor.wait_for_function(REMOTE_PLAYING, timeout=45000)
            patient.wait_for_function(REMOTE_PLAYING, timeout=45000)
            d, p = doctor.evaluate(REMOTE_TRACKS), patient.evaluate(REMOTE_TRACKS)
            assert d == {"audio": 1, "video": 1}, f"doctor received {d}"
            assert p == {"audio": 1, "video": 1}, f"patient received {p}"
            time.sleep(1.5)
            shot(doctor, "4-doctor-call")
            shot(patient, "5-patient-call")
            return f"doctor recv {d}, patient recv {p}"

        def patient_sees_badge():
            label = patient.get_by_test_id("peer-label")
            expect(label).to_contain_text("Verified Doctor")
            if dev_mode:
                expect(label).to_contain_text(pro_name)
            return label.inner_text()

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
            # The badge survives the refresh (it comes back with "joined").
            expect(patient.get_by_test_id("peer-label")).to_contain_text(
                "Verified Doctor"
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

        def help_record_counts_the_call():
            expected = str(helped_before["count"] + 1)
            record = doctor.get_by_test_id("help-record")
            expect(record.get_by_test_id("help-count")).to_have_text(
                expected, timeout=15000
            )
            expect(record.get_by_text("Smoke Test").first).to_be_visible()
            shot(doctor, "7-help-record")
            return f"helped {expected}"

        def patient_signs_in_and_finds_chat():
            patient.get_by_role("link", name="Sign in to save chats").click()
            dev_sign_in(patient, f"patient-{run_id}@smoke.test", "Smoke Patient")
            patient.get_by_role("button", name=re.compile(r"^Patient")).click()
            # Back to the chat the sign-in link came from.
            patient.wait_for_url(f"{base}/patient/", timeout=15000)
            patient.get_by_role("button", name="My chats").click()
            saved = patient.get_by_role("list", name="Saved chats")
            expect(saved.get_by_text("I burned my hand on the stove")).to_be_visible(
                timeout=15000
            )
            shot(patient, "6-my-chats")

        steps = [
            ("doctor goes online", professional_goes_online),
            ("patient chat gets a model reply", patient_chats),
            ("urgent Nepali message shows emergency banner", urgent_nepali),
            ("patient requests a doctor (no login)", request_doctor),
            ("professional sees patient + shared chat", doctor_sees_patient),
            ("video call connects both ways (audio + video)", call_connects),
            ("patient sees the verified badge", patient_sees_badge),
            ("mute and camera-off toggle the real tracks", controls),
            ("patient refresh mid-call re-establishes the call", refresh_rejoins),
            ("hang up ends both sides, chat kept", hang_up),
            ("help record counts the call, with the name", help_record_counts_the_call),
        ]
        if dev_mode:
            steps = [
                (
                    "professional signs in and applies with documents",
                    professional_applies,
                ),
                ("admin reviews documents and approves", admin_approves),
                *steps,
                (
                    "patient signs in; chat appears in My chats",
                    patient_signs_in_and_finds_chat,
                ),
            ]
            steps[2] = ("verified professional goes online", professional_goes_online)

        try:
            for name, fn in steps:
                step(name, fn)
        except Exception:
            # Show what each side was looking at when the step failed.
            for role, page in (
                ("doctor", doctor),
                ("patient", patient),
                ("admin", admin),
            ):
                try:
                    screen = page.evaluate(
                        "() => document.querySelector('main')?.innerText ?? ''"
                    )
                except Exception as exc:
                    screen = f"(unavailable: {exc})"
                print(
                    f"\n--- {role} screen at failure ({page.url}) ---\n{screen[:300]}"
                )
                try:
                    diag = page.evaluate(CALL_DIAGNOSTICS)
                    for i, pc in enumerate(diag["pcs"]):
                        print(f"    {role} peer#{i}: {pc}")
                    for i, video in enumerate(diag["videos"]):
                        print(f"    {role} video#{i}: {video}")
                except Exception as exc:
                    print(f"    {role} diagnostics unavailable: {exc}")
                try:
                    for line in page.evaluate("() => window.__media ?? []")[-15:]:
                        print(f"    {role} media: {line[:300]}")
                except Exception as exc:
                    print(f"    {role} media log unavailable: {exc}")
                for line in logs[role][-10:]:
                    print(f"    {role} console: {line[:220]}")
        finally:
            for b in {id(b): b for b in browsers.values()}.values():
                b.close()

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
