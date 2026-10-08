package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// loc finds elements: a JavaScript expression that evaluates to an array of
// elements. The zero value is the whole document.
type loc struct{ js string }

var doc loc

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// root is the element that a nested lookup searches in.
func (l loc) root() string {
	if l.js == "" {
		return "undefined"
	}
	return "((" + l.js + ")[0] ?? null)"
}

func (l loc) finder(name string, args ...string) loc {
	return loc{"__smoke." + name + "(" + strings.Join(append([]string{l.root()}, args...), ", ") + ")"}
}

// Role finds elements by role and accessible name (containing the name,
// case-insensitive; an empty name matches any).
func (l loc) Role(role, name string) loc {
	if name == "" {
		return l.finder("role", quote(role), `"any"`, "null")
	}
	return l.finder("role", quote(role), `"contains"`, quote(name))
}

// RoleExact matches the whole accessible name.
func (l loc) RoleExact(role, name string) loc {
	return l.finder("role", quote(role), `"exact"`, quote(name))
}

// RolePrefix matches the start of the accessible name.
func (l loc) RolePrefix(role, name string) loc {
	return l.finder("role", quote(role), `"prefix"`, quote(name))
}

// Label finds form fields by their label.
func (l loc) Label(text string) loc { return l.finder("label", quote(text)) }

// Text finds the innermost elements showing a text.
func (l loc) Text(text string) loc { return l.finder("text", quote(text)) }

// TestID finds elements by data-testid.
func (l loc) TestID(id string) loc { return l.finder("testid", quote(id)) }

// CSS finds elements by a CSS selector.
func (l loc) CSS(selector string) loc { return l.finder("css", quote(selector)) }

// Has keeps the elements that contain a text.
func (l loc) Has(text string) loc { return loc{"__smoke.has(" + l.js + ", " + quote(text) + ")"} }

// Last keeps the last element.
func (l loc) Last() loc { return loc{"(" + l.js + ").slice(-1)"} }

// tab is one browser tab: a patient, a professional or the admin.
type tab struct {
	role    string
	ctx     context.Context
	close   context.CancelFunc
	mu      sync.Mutex
	console []string // recent console lines
	errors  []string // console errors and uncaught exceptions
}

var tagCounter atomic.Int64

// openTab opens a tab with its own cookies in a running browser.
func openTab(browser context.Context, role string) (*tab, error) {
	ctx, cancel := chromedp.NewContext(browser, chromedp.WithNewBrowserContext())
	if err := chromedp.Do(ctx, chromedp.EmulateViewport(1280, 800)); err != nil {
		cancel()
		return nil, fmt.Errorf("open the %s tab: %w", role, err)
	}
	_, err := chromedp.Call(ctx, page.AddScriptToEvaluateOnNewDocument,
		page.AddScriptToEvaluateOnNewDocumentParams{Source: finders + "\n" + pcSpy})
	if err != nil {
		cancel()
		return nil, err
	}
	t := &tab{role: role, ctx: ctx, close: cancel}
	messages := chromedp.Console(ctx)
	go func() {
		for m, err := range messages {
			if err != nil {
				return
			}
			line := fmt.Sprintf("[%s] %s", m.Type, m.Text)
			t.mu.Lock()
			t.console = append(t.console, line)
			if len(t.console) > 20 {
				t.console = t.console[1:]
			}
			if m.IsException() || string(m.Type) == "error" {
				t.errors = append(t.errors, m.Text)
			}
			t.mu.Unlock()
		}
	}()
	return t, nil
}

func (t *tab) goTo(url string) error { return chromedp.Do(t.ctx, chromedp.Navigate(url)) }

func (t *tab) reload() error { return chromedp.Do(t.ctx, chromedp.Reload()) }

func (t *tab) setCookie(name, value, url string) error {
	_, err := chromedp.Call(t.ctx, network.SetCookie, network.SetCookieParams{Name: name, Value: value, URL: url})
	return err
}

// eval runs JavaScript and decodes its value.
func eval[T any](t *tab, js string) (T, error) {
	return chromedp.Run(t.ctx, chromedp.Evaluate[T](js, chromedp.EvalAwaitPromise))
}

// waitTrue polls a JavaScript condition until it holds.
func (t *tab) waitTrue(js, what string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		ok, err := eval[bool](t, "!!("+js+")")
		if err == nil && ok {
			return nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("%s: %w", what, err)
			}
			return fmt.Errorf("timed out after %s waiting for %s", timeout, what)
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func (t *tab) visible(l loc, what string, timeout time.Duration) error {
	return t.waitTrue("("+l.js+").some(__smoke.visible)", what+" to be visible", timeout)
}

func (t *tab) count(l loc, n int, what string, timeout time.Duration) error {
	return t.waitTrue(fmt.Sprintf("(%s).length === %d", l.js, n), fmt.Sprintf("%d of %s", n, what), timeout)
}

func (t *tab) hasText(l loc, text, what string, timeout time.Duration) error {
	return t.waitTrue("("+l.js+").some((el) => __smoke.norm(el.innerText).includes("+quote(text)+"))",
		fmt.Sprintf("%s to contain %q", what, text), timeout)
}

func (t *tab) textIs(l loc, text, what string, timeout time.Duration) error {
	return t.waitTrue("("+l.js+").some((el) => __smoke.norm(el.innerText) === "+quote(text)+")",
		fmt.Sprintf("%s to be %q", what, text), timeout)
}

func (t *tab) path(path string, timeout time.Duration) error {
	return t.waitTrue("location.pathname === "+quote(path), "the page "+path, timeout)
}

func (t *tab) innerText(l loc) (string, error) {
	return eval[string](t, "(("+l.js+").find(__smoke.visible) ?? {}).innerText ?? ''")
}

// tag marks the first matching element (visible ones only, if asked) so
// that a CSS selector can reach it, and returns the selector.
func (t *tab) tag(l loc, mustBeVisible bool, what string, timeout time.Duration) (string, error) {
	id := fmt.Sprintf("s%d", tagCounter.Add(1))
	pick := "(" + l.js + ")[0]"
	if mustBeVisible {
		pick = "(" + l.js + ").find(__smoke.visible)"
	}
	js := fmt.Sprintf(`(() => { const el = %s; if (!el) return false; el.setAttribute("data-smoke", %q); return true; })()`, pick, id)
	if err := t.waitTrue(js, what, timeout); err != nil {
		return "", err
	}
	return fmt.Sprintf(`[data-smoke="%s"]`, id), nil
}

// click clicks the first visible match with the mouse, as a person would.
func (t *tab) click(l loc, what string) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		var selector string
		if selector, err = t.tag(l, true, what, 15*time.Second); err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(t.ctx, 5*time.Second)
		err = chromedp.Do(ctx, chromedp.Click(chromedp.CSS(selector), chromedp.NodeVisible))
		cancel()
		if err == nil {
			return nil
		}
		// The page re-rendered between finding and clicking: find it again.
	}
	return fmt.Errorf("click %s: %w", what, err)
}

// fill types into a field, replacing what was there.
func (t *tab) fill(l loc, text, what string) error {
	if err := t.click(l, what); err != nil {
		return err
	}
	clear := `(() => {
	  const el = document.activeElement;
	  const proto = el instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
	  Object.getOwnPropertyDescriptor(proto, "value").set.call(el, "");
	  el.dispatchEvent(new Event("input", { bubbles: true }));
	  return true;
	})()`
	if _, err := eval[bool](t, clear); err != nil {
		return fmt.Errorf("clear %s: %w", what, err)
	}
	_, err := chromedp.Call(t.ctx, input.InsertText, input.InsertTextParams{Text: text})
	return err
}

// pressEnter presses Enter in the focused field, like a person: key down
// with its character, then key up, so a form submits.
func (t *tab) pressEnter() error { return chromedp.Do(t.ctx, chromedp.KeyEvent("\r")) }

// upload sets a file on a (possibly hidden) file input.
func (t *tab) upload(l loc, path, what string) error {
	selector, err := t.tag(l, false, what, 15*time.Second)
	if err != nil {
		return err
	}
	return chromedp.Do(t.ctx, chromedp.SetUploadFiles(chromedp.CSS(selector), []string{path}, chromedp.NodeReady))
}

func (t *tab) screenshot() ([]byte, error) {
	return chromedp.Run(t.ctx, chromedp.FullScreenshot(100))
}

func (t *tab) recentConsole() ([]string, []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.console...), append([]string(nil), t.errors...)
}

// errStep marks a failed check inside a step.
var errStep = errors.New("check failed")
