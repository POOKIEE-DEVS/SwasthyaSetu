package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// Gradio calls an API endpoint of a Gradio app (model-space/app.py) over
// Gradio's HTTP call API:
//
//	POST {app}/gradio_api/call/{name}  {"data": [...]}   -> {"event_id": "..."}
//	GET  {app}/gradio_api/call/{name}/{event_id}          -> server-sent events,
//	     ending with "event: complete" and "data: [outputs...]"
//
// The app is a Hugging Face Space id ("user/space") or the URL of any running
// copy, such as a Colab https://….gradio.live link. Where it lives and its
// API prefix are looked up once and reused; Reset forgets them, for when the
// app restarted somewhere else.
type Gradio struct {
	source string
	token  string
	client *http.Client
	hfAPI  string // https://huggingface.co; tests point it elsewhere

	mu   sync.Mutex
	base string // e.g. "https://user-space.hf.space/gradio_api/"
}

// NewGradio returns a client for a Space id or app URL. The token is a
// Hugging Face read token for a private Space; it is only ever sent to
// Hugging Face, never to other hosts such as a gradio.live link.
func NewGradio(source, token string, client *http.Client) *Gradio {
	if client == nil {
		client = http.DefaultClient
	}
	return &Gradio{
		source: strings.TrimSpace(source),
		token:  strings.TrimSpace(token),
		client: client,
		hfAPI:  "https://huggingface.co",
	}
}

// Reset forgets the resolved app address.
func (g *Gradio) Reset() {
	g.mu.Lock()
	g.base = ""
	g.mu.Unlock()
}

// Call runs the named endpoint and returns its first output.
func (g *Gradio) Call(ctx context.Context, name string, args ...any) (json.RawMessage, error) {
	base, err := g.resolve(ctx)
	if err != nil {
		return nil, err
	}
	if args == nil {
		args = []any{}
	}
	body, err := json.Marshal(map[string]any{"data": args})
	if err != nil {
		return nil, err
	}
	endpoint := base + "call/" + url.PathEscape(name)

	var queued struct {
		EventID string `json:"event_id"`
	}
	if err := g.doJSON(ctx, http.MethodPost, endpoint, body, &queued); err != nil {
		return nil, err
	}
	if queued.EventID == "" {
		return nil, errors.New("gradio: no event id in the response")
	}

	resp, err := g.send(ctx, http.MethodGet, endpoint+"/"+url.PathEscape(queued.EventID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return readResult(resp.Body)
}

// readResult reads the event stream until the call completes or fails.
// Heartbeats and progress events are skipped.
func readResult(r io.Reader) (json.RawMessage, error) {
	reader := bufio.NewReader(r)
	var event string
	var data strings.Builder
	for {
		line, readErr := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if field, value, ok := strings.Cut(line, ":"); ok {
			value = strings.TrimPrefix(value, " ")
			switch field {
			case "event":
				event = value
			case "data":
				if data.Len() > 0 {
					data.WriteByte('\n')
				}
				data.WriteString(value)
			}
		}
		// A blank line ends an event (so does the end of the stream).
		if (line == "" || readErr != nil) && event != "" {
			switch event {
			case "complete":
				var outputs []json.RawMessage
				if err := json.Unmarshal([]byte(data.String()), &outputs); err != nil {
					return nil, fmt.Errorf("gradio: unreadable result: %w", err)
				}
				if len(outputs) == 0 {
					return nil, errors.New("gradio: empty result")
				}
				return outputs[0], nil
			case "error":
				return nil, fmt.Errorf("gradio: the app reported an error: %.300s", data.String())
			}
			event = ""
			data.Reset()
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil, errors.New("gradio: the stream ended without a result")
			}
			return nil, readErr
		}
	}
}

// resolve finds the app's API base URL, once.
func (g *Gradio) resolve(ctx context.Context) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.base != "" {
		return g.base, nil
	}
	root, err := g.appRoot(ctx)
	if err != nil {
		return "", err
	}
	var config struct {
		APIPrefix string `json:"api_prefix"`
	}
	if err := g.doJSON(ctx, http.MethodGet, root+"config", nil, &config); err != nil {
		return "", err
	}
	base := root
	if prefix := strings.Trim(config.APIPrefix, "/"); prefix != "" {
		base += prefix + "/"
	}
	g.base = base
	return base, nil
}

// appRoot is the app's URL, with a trailing slash. A Space id is looked up
// on Hugging Face.
func (g *Gradio) appRoot(ctx context.Context) (string, error) {
	if strings.HasPrefix(g.source, "http://") || strings.HasPrefix(g.source, "https://") {
		return strings.TrimRight(g.source, "/") + "/", nil
	}
	owner, name, ok := strings.Cut(g.source, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return "", fmt.Errorf("gradio: %q is neither a Space id (user/space) nor a URL", g.source)
	}
	var info struct {
		Host string `json:"host"`
	}
	endpoint := g.hfAPI + "/api/spaces/" + url.PathEscape(owner) + "/" + url.PathEscape(name)
	if err := g.doJSON(ctx, http.MethodGet, endpoint, nil, &info); err != nil {
		return "", fmt.Errorf("gradio: could not find Space %s (a private Space needs HF_TOKEN): %w", g.source, err)
	}
	if info.Host == "" {
		return "", fmt.Errorf("gradio: Space %s has no address; is it running?", g.source)
	}
	return strings.TrimRight(info.Host, "/") + "/", nil
}

func (g *Gradio) doJSON(ctx context.Context, method, endpoint string, body []byte, out any) error {
	resp, err := g.send(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out); err != nil {
		return fmt.Errorf("gradio: unreadable response from %s: %w", redact(endpoint), err)
	}
	return nil
}

// send makes a request and fails on any status other than 200.
func (g *Gradio) send(ctx context.Context, method, endpoint string, body []byte) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", "swasthyasetu-backend")
	if g.token != "" && isHuggingFace(req.URL) {
		req.Header.Set("Authorization", "Bearer "+g.token)
		// What gradio_client sends; Spaces accept either.
		req.Header.Set("X-HF-Authorization", "Bearer "+g.token)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		resp.Body.Close()
		return nil, fmt.Errorf("gradio: %s %s: %s %s", method, redact(endpoint), resp.Status, strings.TrimSpace(string(snippet)))
	}
	return resp, nil
}

func isHuggingFace(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	return host == "huggingface.co" || strings.HasSuffix(host, ".huggingface.co") || strings.HasSuffix(host, ".hf.space")
}

// redact keeps logs to the host and path.
func redact(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "the model server"
	}
	return u.Scheme + "://" + u.Host + u.Path
}
