package realtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/config"
)

// ICE servers are handed to each browser before it opens its peer
// connection.
//
// STUN alone connects two laptops only when they can reach each other more
// or less directly. Across different networks (venue Wi-Fi, mobile hotspots,
// carrier NAT) the call needs TURN, a relay the media flows through. The
// media stays encrypted end to end (DTLS-SRTP), but it does pass the relay.
//
// Web hosts like Render expose no UDP ports, so TURN is a hosted service:
// - Cloudflare: short-lived credentials are minted per call through their
//   API, so no long-lived secret reaches a browser.
// - Static credentials from any TURN provider (e.g. ExpressTURN).

// IceServer is one entry of RTCPeerConnection's iceServers option.
type IceServer struct {
	URLs       []string `json:"urls"`
	Username   *string  `json:"username"`
	Credential *string  `json:"credential"`
}

// ICE builds the ICE server list.
type ICE struct {
	cfg           *config.Config
	client        *http.Client
	cloudflareAPI string // https://rtc.live.cloudflare.com; tests point it elsewhere
	log           *slog.Logger
}

// NewICE returns an ICE server source for the configuration.
func NewICE(cfg *config.Config, log *slog.Logger) *ICE {
	return &ICE{
		cfg:           cfg,
		client:        &http.Client{Timeout: 10 * time.Second},
		cloudflareAPI: "https://rtc.live.cloudflare.com",
		log:           log,
	}
}

// Servers returns STUN always, plus TURN when configured. If Cloudflare
// fails, it falls back to the static relay or STUN only rather than failing
// the call: same-network calls still connect, and the log says why others
// may not.
func (i *ICE) Servers(ctx context.Context) []IceServer {
	if i.cfg.CloudflareTURN() {
		servers, err := i.cloudflare(ctx)
		if err == nil {
			return servers
		}
		i.log.Error("cloudflare TURN credential request failed", "error", err)
	}
	servers := []IceServer{}
	if len(i.cfg.STUNURLs) > 0 {
		servers = append(servers, IceServer{URLs: i.cfg.STUNURLs})
	}
	if len(i.cfg.TURNURLs) > 0 {
		servers = append(servers, IceServer{
			URLs:       i.cfg.TURNURLs,
			Username:   optional(i.cfg.TURNUsername),
			Credential: optional(i.cfg.TURNCredential),
		})
	}
	return servers
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (i *ICE) cloudflare(ctx context.Context) ([]IceServer, error) {
	endpoint := fmt.Sprintf("%s/v1/turn/keys/%s/credentials/generate-ice-servers",
		i.cloudflareAPI, url.PathEscape(i.cfg.CloudflareTURNKeyID))
	body, _ := json.Marshal(map[string]int{"ttl": i.cfg.TURNCredentialTTL})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+i.cfg.CloudflareTURNAPIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := i.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("status %s", resp.Status)
	}
	var result struct {
		IceServers []struct {
			URLs       json.RawMessage `json:"urls"`
			Username   *string         `json:"username"`
			Credential *string         `json:"credential"`
		} `json:"iceServers"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("unreadable response: %w", err)
	}
	servers := []IceServer{}
	for _, entry := range result.IceServers {
		var urls []string
		if err := json.Unmarshal(entry.URLs, &urls); err != nil {
			var one string
			if err := json.Unmarshal(entry.URLs, &one); err != nil {
				return nil, fmt.Errorf("unreadable urls: %s", entry.URLs)
			}
			urls = []string{one}
		}
		if urls = dropPort53(urls); len(urls) > 0 {
			servers = append(servers, IceServer{URLs: urls, Username: entry.Username, Credential: entry.Credential})
		}
	}
	return servers, nil
}

// dropPort53 removes port-53 URLs: browsers block that port, and those
// candidates just time out.
func dropPort53(urls []string) []string {
	var out []string
	for _, u := range urls {
		if !strings.Contains(u, ":53?") && !strings.HasSuffix(u, ":53") {
			out = append(out, u)
		}
	}
	return out
}
