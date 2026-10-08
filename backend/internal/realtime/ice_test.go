package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/config"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/logging"
)

const cloudflareResponse = `{"iceServers": [
	{"urls": ["stun:stun.cloudflare.com:3478", "stun:stun.cloudflare.com:53"]},
	{"urls": ["turn:turn.cloudflare.com:3478?transport=udp",
	          "turn:turn.cloudflare.com:53?transport=udp",
	          "turns:turn.cloudflare.com:443?transport=tcp"],
	 "username": "u", "credential": "c"}
]}`

func TestStunOnlyByDefault(t *testing.T) {
	servers := NewICE(config.Defaults(), logging.Discard()).Servers(context.Background())
	if len(servers) != 1 {
		t.Fatalf("servers %+v", servers)
	}
	for _, u := range servers[0].URLs {
		if !strings.HasPrefix(u, "stun:") {
			t.Errorf("not STUN: %s", u)
		}
	}
	// The JSON always has username and credential, null when unset.
	data, _ := json.Marshal(servers[0])
	if !strings.Contains(string(data), `"username":null,"credential":null`) {
		t.Errorf("json %s", data)
	}
}

func TestStaticTURNCredentials(t *testing.T) {
	cfg := config.Defaults()
	cfg.TURNURLs = []string{"turn:relay.example:3478"}
	cfg.TURNUsername, cfg.TURNCredential = "demo", "secret"
	servers := NewICE(cfg, logging.Discard()).Servers(context.Background())
	if len(servers) != 2 || !strings.HasPrefix(servers[0].URLs[0], "stun:") {
		t.Fatalf("servers %+v", servers)
	}
	turn := servers[1]
	if turn.URLs[0] != "turn:relay.example:3478" || *turn.Username != "demo" || *turn.Credential != "secret" {
		t.Fatalf("turn %+v", turn)
	}
}

func cloudflareICE(t *testing.T, handler http.HandlerFunc) *ICE {
	cfg := config.Defaults()
	cfg.CloudflareTURNKeyID, cfg.CloudflareTURNAPIToken = "key", "token"
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	ice := NewICE(cfg, logging.Discard())
	ice.cloudflareAPI = srv.URL
	return ice
}

func TestCloudflareCredentialsDropPort53(t *testing.T) {
	var auth, path string
	var body map[string]int
	ice := cloudflareICE(t, func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(cloudflareResponse))
	})
	servers := ice.Servers(context.Background())

	if auth != "Bearer token" || !strings.Contains(path, "/keys/key/") || body["ttl"] != 86400 {
		t.Fatalf("request: %q %q %v", auth, path, body)
	}
	for _, s := range servers {
		for _, u := range s.URLs {
			if strings.Contains(u, ":53") {
				t.Errorf("port 53 kept: %s", u)
			}
		}
	}
	if len(servers) != 2 || *servers[1].Username != "u" || len(servers[1].URLs) != 2 {
		t.Fatalf("servers %+v", servers)
	}
}

func TestCloudflareFailureFallsBackToStun(t *testing.T) {
	ice := cloudflareICE(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	servers := ice.Servers(context.Background())
	if len(servers) != 1 || len(servers[0].URLs) != 2 || !strings.HasPrefix(servers[0].URLs[1], "stun:") {
		t.Fatalf("servers %+v", servers)
	}
}

func TestCloudflareSingleURLString(t *testing.T) {
	ice := cloudflareICE(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"iceServers": [{"urls": "turn:t.example:3478", "username": "u", "credential": "c"}]}`))
	})
	servers := ice.Servers(context.Background())
	if len(servers) != 1 || servers[0].URLs[0] != "turn:t.example:3478" {
		t.Fatalf("servers %+v", servers)
	}
}
