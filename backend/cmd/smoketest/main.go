// Command smoketest runs the whole demo in real Chrome windows and reports
// each step. Run it before every rehearsal and after every deploy.
//
// A professional, an admin and a patient work side by side, with Chrome's
// fake camera and microphone:
//
//	professional signs in -> applies as a doctor with documents
//	-> admin approves -> professional goes online
//	-> patient chats (English, then urgent Nepali) -> requests a doctor
//	-> professional sees them live, with the shared chat -> accepts
//	-> two-way audio+video, patient sees "Verified Doctor" -> mute / camera off
//	-> patient refreshes mid-call and the call re-establishes -> hang up
//	-> the professional's help record counts the call, with the patient's name
//	-> patient signs in and finds the conversation in "My chats"
//
// Locally, run the server with DEV_LOGIN=true and ADMIN_EMAILS including
// admin@smoke.test; the test then signs everyone in without Google:
//
//	go run ./cmd/smoketest http://localhost:8000
//
// Against the deployed site (Google sign-in only), pass the session cookie
// of an already verified professional; the sign-up, review and patient
// sign-in steps are skipped:
//
//	go run ./cmd/smoketest https://your-app.onrender.com --pro-session <value of the swasthya_session cookie>
//
// Options: --shots DIR saves screenshots, --headed shows the browsers,
// --chrome PATH picks the Chrome binary. Exits 1 if any step fails, 2 if it
// cannot run. Against the real model the first reply can take a while (cold
// start); the chat step waits up to 3 minutes.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	adminEmail    = "admin@smoke.test"
	sessionCookie = "swasthya_session"
)

// A real 1x1 PNG, used for every uploaded document.
var png, _ = base64.StdEncoding.DecodeString(
	"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==")

type options struct {
	base       string
	shots      string
	headed     bool
	proSession string
	chrome     string
}

func parseArgs(args []string) (options, error) {
	var o options
	for i := 0; i < len(args); i++ {
		arg := args[i]
		value := func() (string, error) {
			if name, v, ok := strings.Cut(arg, "="); ok && strings.HasPrefix(name, "--") {
				return v, nil
			}
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s needs a value", arg)
			}
			i++
			return args[i], nil
		}
		var err error
		switch name, _, _ := strings.Cut(arg, "="); name {
		case "--shots":
			o.shots, err = value()
		case "--pro-session":
			o.proSession, err = value()
		case "--chrome":
			o.chrome, err = value()
		case "--headed":
			o.headed = true
		case "-h", "--help":
			return o, errors.New("help")
		default:
			if strings.HasPrefix(arg, "-") || o.base != "" {
				return o, fmt.Errorf("unexpected argument %q", arg)
			}
			o.base = strings.TrimRight(arg, "/")
		}
		if err != nil {
			return o, err
		}
	}
	if o.base == "" {
		return o, errors.New("the site's address is missing")
	}
	return o, nil
}

const usage = `usage: go run ./cmd/smoketest <base-url> [--pro-session COOKIE] [--shots DIR] [--headed] [--chrome PATH]

  base-url       e.g. http://localhost:8000 or https://your-app.onrender.com
  --pro-session  session cookie of a verified professional (for the deployed site)
  --shots DIR    save screenshots there
  --headed       show the browsers
  --chrome PATH  the Chrome or Chromium binary (found automatically otherwise)`

func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		if err.Error() != "help" {
			fmt.Fprintln(os.Stderr, err)
		}
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	os.Exit(run(opts))
}

// writeFakeCamera writes a small Y4M clip (320x240, moving gradient) for
// Chrome to play as the camera. Chrome's built-in fake camera sometimes ends
// its own track a moment after it starts; a file-backed camera doesn't.
func writeFakeCamera(path string) error {
	const width, height, frames = 320, 240, 20
	quarter := width * height / 4
	var b strings.Builder
	fmt.Fprintf(&b, "YUV4MPEG2 W%d H%d F15:1 Ip A1:1 C420jpeg\n", width, height)
	for n := range frames {
		b.WriteString("FRAME\n")
		luma := make([]byte, width*height)
		for y := range height {
			for x := range width {
				luma[y*width+x] = byte((x + y + n*8) % 256)
			}
		}
		b.Write(luma)
		b.Write(bytesOf(128, quarter))
		b.Write(bytesOf(byte((n*12)%256), quarter))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func bytesOf(v byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = v
	}
	return out
}

type result struct {
	name   string
	ok     bool
	detail string
}

func run(o options) int {
	runID := strconv.FormatInt(time.Now().UnixNano(), 36)
	runID = runID[len(runID)-6:]
	proName := "Dr. Smoke " + runID

	if o.shots != "" {
		if err := os.MkdirAll(o.shots, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	}
	dir, err := os.MkdirTemp("", "swasthyasetu-smoke-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	defer os.RemoveAll(dir)
	camera := filepath.Join(dir, "fake-camera.y4m")
	document := filepath.Join(dir, "doc.png")
	if err := writeFakeCamera(camera); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if err := os.WriteFile(document, png, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	devMode, err := hasDevLogin(o.base)
	if err != nil {
		fmt.Fprintln(os.Stderr, "can't reach the site:", err)
		return 2
	}
	if !devMode && o.proSession == "" {
		fmt.Println("This server has no development login. Pass --pro-session with the\n" +
			"swasthya_session cookie of a verified professional (see --help).")
		return 2
	}

	// One Chrome process per call participant, like two laptops. The admin
	// shares the professional's Chrome (never uses the camera) in its own
	// cookie jar.
	launch := func() (context.Context, context.CancelFunc, error) {
		flags := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("use-fake-device-for-media-stream", true),
			chromedp.Flag("use-fake-ui-for-media-stream", true),
			chromedp.Flag("autoplay-policy", "no-user-gesture-required"),
			chromedp.Flag("use-file-for-fake-video-capture", camera),
		)
		if o.chrome != "" {
			flags = append(flags, chromedp.ExecPath(o.chrome))
		}
		allocator, cancelAllocator := chromedp.NewExecAllocator(context.Background(), flags...)
		var contextOptions []chromedp.ContextOption
		if o.headed {
			contextOptions = append(contextOptions, chromedp.WithVisibleWindow())
		}
		browser, cancelBrowser := chromedp.NewContext(allocator, contextOptions...)
		cancel := func() { cancelBrowser(); cancelAllocator() }
		if err := chromedp.Do(browser); err != nil { // starts Chrome
			cancel()
			return nil, nil, fmt.Errorf("start Chrome: %w", err)
		}
		return browser, cancel, nil
	}
	doctorBrowser, closeDoctor, err := launch()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	defer closeDoctor()
	patientBrowser, closePatient, err := launch()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	defer closePatient()

	doctor, err1 := openTab(doctorBrowser, "doctor")
	patient, err2 := openTab(patientBrowser, "patient")
	admin, err3 := openTab(doctorBrowser, "admin")
	if err := errors.Join(err1, err2, err3); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	tabs := []*tab{doctor, patient, admin}
	defer func() {
		for _, t := range tabs {
			t.close()
		}
	}()

	shot := func(t *tab, name string) {
		if o.shots == "" {
			return
		}
		if img, err := t.screenshot(); err == nil {
			_ = os.WriteFile(filepath.Join(o.shots, name+".png"), img, 0o644)
		}
	}
	devSignIn := func(t *tab, email, name string) error {
		if err := t.fill(doc.Label("Development email"), email, "the development email"); err != nil {
			return err
		}
		if err := t.fill(doc.Label("Development name"), name, "the development name"); err != nil {
			return err
		}
		return t.click(doc.Role("button", "Sign in for testing"), "Sign in for testing")
	}
	const short, long = 15 * time.Second, 180 * time.Second
	helpedBefore := 0

	professionalApplies := func() (string, error) {
		if err := doctor.goTo(o.base + "/doctor/"); err != nil {
			return "", err
		}
		if err := doctor.visible(doc.Text("For medical professionals"), "the professionals' page", short); err != nil {
			return "", err
		}
		if err := doctor.click(doc.RoleExact("link", "Sign in").Last(), "Sign in"); err != nil {
			return "", err
		}
		if err := devSignIn(doctor, "doctor-"+runID+"@smoke.test", proName); err != nil {
			return "", err
		}
		// Exact start: the student card mentions "a doctor" too.
		if err := doctor.click(doc.RolePrefix("button", "Doctor"), "the Doctor role"); err != nil {
			return "", err
		}
		if err := doctor.visible(doc.Role("heading", "Get verified"), "the verification form", short); err != nil {
			return "", err
		}
		for label, value := range map[string]string{
			"Full name, as on your citizenship": proName,
			"Phone number":                      "9841234567",
			"Citizenship number":                "27-01-71-12345",
			"Issuing district":                  "Kathmandu",
			"registration number":               "12345",
		} {
			if err := doctor.fill(doc.Label(label), value, label); err != nil {
				return "", err
			}
		}
		for _, label := range []string{"Citizenship: front", "Citizenship: back", "NMC registration certificate"} {
			if err := doctor.upload(doc.Label(label), document, label); err != nil {
				return "", err
			}
		}
		if err := doctor.click(doc.Role("checkbox", ""), "the consent box"); err != nil {
			return "", err
		}
		shot(doctor, "0-apply-form")
		if err := doctor.click(doc.Role("button", "Submit for review"), "Submit for review"); err != nil {
			return "", err
		}
		return "", doctor.visible(doc.Text("Waiting for review"), "Waiting for review", short)
	}

	adminApproves := func() (string, error) {
		if err := admin.goTo(o.base + "/account/?next=/admin/"); err != nil {
			return "", err
		}
		if err := devSignIn(admin, adminEmail, "Smoke Admin"); err != nil {
			return "", err
		}
		if err := admin.path("/admin/", short); err != nil {
			return "", err
		}
		card := doc.TestID("application").Has(proName)
		if err := admin.visible(card, "the application card", short); err != nil {
			return "", err
		}
		// Every document is viewable by the admin.
		if err := admin.count(card.Role("img", ""), 3, "document images", short); err != nil {
			return "", err
		}
		shot(admin, "1-admin-review")
		if err := admin.click(card.Role("button", "Approve"), "Approve"); err != nil {
			return "", err
		}
		return "", admin.count(card, 0, "application cards left", short)
	}

	professionalGoesOnline := func() (string, error) {
		if devMode {
			if err := doctor.click(doc.Role("button", "Check again"), "Check again"); err != nil {
				return "", err
			}
		} else {
			if err := doctor.setCookie(sessionCookie, o.proSession, o.base); err != nil {
				return "", err
			}
			if err := doctor.goTo(o.base + "/doctor/"); err != nil {
				return "", err
			}
		}
		if err := doctor.visible(doc.Text("Verified Doctor"), "Verified Doctor", short); err != nil {
			return "", err
		}
		if err := doctor.click(doc.Role("button", "Go online"), "Go online"); err != nil {
			return "", err
		}
		helpCount := doc.TestID("help-count")
		if err := doctor.visible(helpCount, "the help count", short); err != nil {
			return "", err
		}
		text, err := doctor.innerText(helpCount)
		if err != nil {
			return "", err
		}
		if helpedBefore, err = strconv.Atoi(strings.TrimSpace(text)); err != nil {
			return "", fmt.Errorf("help count %q: %w", text, err)
		}
		return "", doctor.visible(doc.Text("Online · you'll hear a tone"), "online status", short)
	}

	assistant := doc.CSS(`[data-role="assistant"]`)
	patientChats := func() (string, error) {
		// The landing page's main action opens the chat, no sign-in.
		if err := patient.goTo(o.base + "/"); err != nil {
			return "", err
		}
		if err := patient.click(doc.Role("link", "Get first-aid help"), "Get first-aid help"); err != nil {
			return "", err
		}
		if err := patient.path("/patient/", short); err != nil {
			return "", err
		}
		if err := patient.fill(doc.Label("Message"), "I burned my hand on the stove", "the message box"); err != nil {
			return "", err
		}
		if err := patient.pressEnter(); err != nil {
			return "", err
		}
		if err := patient.visible(assistant, "the model's reply", long); err != nil {
			return "", err
		}
		text, err := patient.innerText(assistant)
		if len([]rune(text)) > 60 {
			text = string([]rune(text)[:60])
		}
		return fmt.Sprintf("reply: %q", text), err
	}

	urgentNepali := func() (string, error) {
		if err := patient.fill(doc.Label("Message"), "मेरो बुबाको छाती दुख्यो", "the message box"); err != nil {
			return "", err
		}
		if err := patient.pressEnter(); err != nil {
			return "", err
		}
		if err := patient.visible(doc.Text("This may be an emergency"), "the emergency banner", long); err != nil {
			return "", err
		}
		if err := patient.count(assistant, 2, "assistant replies", long); err != nil {
			return "", err
		}
		shot(patient, "2-patient-chat")
		return "", nil
	}

	requestDoctor := func() (string, error) {
		if err := patient.click(doc.TestID("talk-to-professional"), "Talk to a professional"); err != nil {
			return "", err
		}
		if err := patient.fill(doc.Label("Your name · तपाईंको नाम"), "Smoke Test", "the patient's name"); err != nil {
			return "", err
		}
		if err := patient.click(doc.Role("button", "Request a doctor"), "Request a doctor"); err != nil {
			return "", err
		}
		return "", patient.visible(doc.Text("Waiting for a doctor to accept"), "the waiting screen", 20*time.Second)
	}

	queueCard := doc.Role("listitem", "").Has("Smoke Test")
	doctorSeesPatient := func() (string, error) {
		if err := doctor.visible(queueCard, "the patient in the queue", short); err != nil {
			return "", err
		}
		if err := doctor.click(queueCard.Text("Read their chat with the assistant"), "Read their chat"); err != nil {
			return "", err
		}
		if err := doctor.visible(queueCard.Text("I burned my hand on the stove"), "the shared chat", short); err != nil {
			return "", err
		}
		shot(doctor, "3-doctor-queue")
		return "", nil
	}

	callConnects := func() (string, error) {
		if err := doctor.click(queueCard.Role("button", "Accept"), "Accept"); err != nil {
			return "", err
		}
		if err := doctor.waitTrue(remotePlaying, "the patient's video at the doctor", 45*time.Second); err != nil {
			return "", err
		}
		if err := patient.waitTrue(remotePlaying, "the doctor's video at the patient", 45*time.Second); err != nil {
			return "", err
		}
		d, err := eval[map[string]int](doctor, remoteTracks)
		if err != nil {
			return "", err
		}
		p, err := eval[map[string]int](patient, remoteTracks)
		if err != nil {
			return "", err
		}
		if d["audio"] != 1 || d["video"] != 1 {
			return "", fmt.Errorf("%w: doctor received %v", errStep, d)
		}
		if p["audio"] != 1 || p["video"] != 1 {
			return "", fmt.Errorf("%w: patient received %v", errStep, p)
		}
		time.Sleep(1500 * time.Millisecond)
		shot(doctor, "4-doctor-call")
		shot(patient, "5-patient-call")
		return fmt.Sprintf("doctor recv {'audio': %d, 'video': %d}, patient recv {'audio': %d, 'video': %d}",
			d["audio"], d["video"], p["audio"], p["video"]), nil
	}

	peerLabel := doc.TestID("peer-label")
	patientSeesBadge := func() (string, error) {
		if err := patient.hasText(peerLabel, "Verified Doctor", "the peer label", short); err != nil {
			return "", err
		}
		if devMode {
			if err := patient.hasText(peerLabel, proName, "the peer label", short); err != nil {
				return "", err
			}
		}
		return patient.innerText(peerLabel)
	}

	controls := func() (string, error) {
		if err := patient.click(doc.Role("button", "Mute microphone"), "Mute microphone"); err != nil {
			return "", err
		}
		if err := patient.visible(doc.Role("button", "Unmute microphone"), "Unmute microphone", short); err != nil {
			return "", err
		}
		muted, err := eval[bool](patient,
			`document.querySelectorAll("video")[1].srcObject.getAudioTracks()[0].enabled === false`)
		if err != nil {
			return "", err
		}
		if !muted {
			return "", fmt.Errorf("%w: the mic track is still enabled after Mute", errStep)
		}
		if err := patient.click(doc.Role("button", "Turn camera off"), "Turn camera off"); err != nil {
			return "", err
		}
		return "", patient.visible(doc.Text("Camera off"), "Camera off", short)
	}

	refreshRejoins := func() (string, error) {
		before, err := eval[string](doctor, streamID)
		if err != nil {
			return "", err
		}
		if err := patient.reload(); err != nil {
			return "", err
		}
		if err := patient.waitTrue(remotePlaying, "the doctor's video after the refresh", 45*time.Second); err != nil {
			return "", err
		}
		newStream := `(() => { const v = document.querySelectorAll("video")[0];
		  return !!v && !!v.srcObject && v.srcObject.id !== ` + quote(before) + ` && v.videoWidth > 0 && v.readyState >= 2; })()`
		if err := doctor.waitTrue(newStream, "a new stream from the patient", 45*time.Second); err != nil {
			return "", err
		}
		// The badge survives the refresh (it comes back with "joined").
		return "", patient.hasText(peerLabel, "Verified Doctor", "the peer label", short)
	}

	hangUp := func() (string, error) {
		if err := doctor.click(doc.Role("button", "End call"), "End call"); err != nil {
			return "", err
		}
		if err := patient.visible(doc.Text("The doctor ended the call."), "the call-ended message", short); err != nil {
			return "", err
		}
		if err := doctor.click(doc.Role("button", "Done"), "Done (doctor)"); err != nil {
			return "", err
		}
		if err := patient.click(doc.Role("button", "Done"), "Done (patient)"); err != nil {
			return "", err
		}
		return "", patient.visible(doc.Text("I burned my hand on the stove"), "the kept chat", short)
	}

	helpRecordCounts := func() (string, error) {
		expected := strconv.Itoa(helpedBefore + 1)
		record := doc.TestID("help-record")
		if err := doctor.textIs(record.TestID("help-count"), expected, "the help count", short); err != nil {
			return "", err
		}
		if err := doctor.visible(record.Text("Smoke Test"), "the patient in the help record", short); err != nil {
			return "", err
		}
		shot(doctor, "7-help-record")
		return "helped " + expected, nil
	}

	patientFindsChat := func() (string, error) {
		if err := patient.click(doc.Role("link", "Sign in to save chats"), "Sign in to save chats"); err != nil {
			return "", err
		}
		if err := devSignIn(patient, "patient-"+runID+"@smoke.test", "Smoke Patient"); err != nil {
			return "", err
		}
		if err := patient.click(doc.RolePrefix("button", "Patient"), "the Patient role"); err != nil {
			return "", err
		}
		// Back to the chat the sign-in link came from.
		if err := patient.path("/patient/", short); err != nil {
			return "", err
		}
		if err := patient.click(doc.Role("button", "My chats"), "My chats"); err != nil {
			return "", err
		}
		saved := doc.Role("list", "Saved chats")
		if err := patient.visible(saved.Text("I burned my hand on the stove"), "the saved chat", short); err != nil {
			return "", err
		}
		shot(patient, "6-my-chats")
		return "", nil
	}

	type step struct {
		name string
		fn   func() (string, error)
	}
	steps := []step{
		{"doctor goes online", professionalGoesOnline},
		{"patient chat gets a model reply", patientChats},
		{"urgent Nepali message shows emergency banner", urgentNepali},
		{"patient requests a doctor (no login)", requestDoctor},
		{"professional sees patient + shared chat", doctorSeesPatient},
		{"video call connects both ways (audio + video)", callConnects},
		{"patient sees the verified badge", patientSeesBadge},
		{"mute and camera-off toggle the real tracks", controls},
		{"patient refresh mid-call re-establishes the call", refreshRejoins},
		{"hang up ends both sides, chat kept", hangUp},
		{"help record counts the call, with the name", helpRecordCounts},
	}
	if devMode {
		steps[0].name = "verified professional goes online"
		steps = append([]step{
			{"professional signs in and applies with documents", professionalApplies},
			{"admin reviews documents and approves", adminApproves},
		}, steps...)
		steps = append(steps, step{"patient signs in; chat appears in My chats", patientFindsChat})
	}

	var results []result
	for _, s := range steps {
		started := time.Now()
		detail, err := s.fn()
		if err != nil {
			message := err.Error()
			if len(message) > 300 {
				message = message[:300]
			}
			results = append(results, result{s.name, false, message})
			report(tabs)
			break
		}
		results = append(results, result{s.name, true,
			strings.TrimSpace(fmt.Sprintf("%s (%.1fs)", detail, time.Since(started).Seconds()))})
	}

	fmt.Println()
	for _, r := range results {
		status := "PASS"
		if !r.ok {
			status = "FAIL"
		}
		fmt.Printf("%s  %s  %s\n", status, r.name, r.detail)
	}
	if skipped := len(steps) - len(results); skipped > 0 {
		fmt.Printf("SKIP  %d later step(s) not run\n", skipped)
	}
	for _, t := range tabs {
		if _, errs := t.recentConsole(); len(errs) > 0 {
			fmt.Printf("\nbrowser console errors (%s):\n", t.role)
			for i, e := range errs {
				if i == 8 {
					break
				}
				fmt.Println("   ", cut(e, 200))
			}
		}
	}
	ok := len(results) == len(steps) && results[len(results)-1].ok
	if ok {
		fmt.Println("\nALL GOOD — ready to demo.")
		return 0
	}
	fmt.Println("\nNOT READY — see failures above.")
	return 1
}

// report shows what each side was looking at when a step failed.
func report(tabs []*tab) {
	for _, t := range tabs {
		location, _ := eval[string](t, "location.href")
		screen, err := eval[string](t, `document.querySelector("main")?.innerText ?? ""`)
		if err != nil {
			screen = "(unavailable: " + err.Error() + ")"
		}
		fmt.Printf("\n--- %s screen at failure (%s) ---\n%s\n", t.role, location, cut(screen, 300))
		if lines, err := eval[[]string](t, callDiagnostics); err == nil {
			for _, line := range lines {
				fmt.Printf("    %s %s\n", t.role, line)
			}
		}
		if media, err := eval[[]string](t, "window.__media ?? []"); err == nil {
			if len(media) > 15 {
				media = media[len(media)-15:]
			}
			for _, line := range media {
				fmt.Printf("    %s media: %s\n", t.role, cut(line, 300))
			}
		}
		console, _ := t.recentConsole()
		if len(console) > 10 {
			console = console[len(console)-10:]
		}
		for _, line := range console {
			fmt.Printf("    %s console: %s\n", t.role, cut(line, 220))
		}
	}
}

func cut(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// hasDevLogin asks the server whether the development login is on.
func hasDevLogin(base string) (bool, error) {
	client := &http.Client{Timeout: 90 * time.Second} // Render's free plan may be waking up
	resp, err := client.Get(base + "/api/v1/auth/me")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var me struct {
		DevLogin bool `json:"dev_login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return false, fmt.Errorf("/api/v1/auth/me: %w", err)
	}
	return me.DevLogin, nil
}
