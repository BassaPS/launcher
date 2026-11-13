package platformService

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var fixtureNow = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)

const fixtureTime = "2026-11-13T12:00:00.000Z"

func statusFixture() map[string]any {
	return map[string]any{
		"overallStatus": "operational", "updatedAt": fixtureTime, "generatedAt": fixtureTime,
		"incidents": []any{},
		"services": []any{map[string]any{"id": "docs", "name": "Documentation", "monitoringType": "http",
			"enabled": true, "status": "operational", "lastCheckedAt": fixtureTime,
			"responseTimeMs": 12, "samples": map[string]any{"24h": 1}}},
	}
}

func releaseFixture() map[string]any {
	asset := map[string]any{"id": 2, "name": "Launcher.zip", "size": 100, "downloads": 0,
		"url": "https://github.com/BassaPS/bassaPSLauncher/releases/download/First/Launcher.zip", "digest": "sha256:" + strings.Repeat("a", 64)}
	return map[string]any{"fetchedAt": fixtureTime, "stale": false,
		"latest": map[string]any{"id": 1, "tag": "First", "version": "v1.1.2", "name": "Launcher 1.1.2",
			"publishedAt": fixtureTime, "body": "", "prerelease": false,
			"url": "https://github.com/BassaPS/bassaPSLauncher/releases/tag/First", "assets": []any{asset}},
		"recommendedAsset": asset,
	}
}

func encoded(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Only tests can inject a server. Public methods always use the fixed endpoints.
func redirectedClient(t *testing.T, serverURL string) *PlatformService {
	t.Helper()
	s := New()
	s.now = func() time.Time { return fixtureNow }
	s.transport.Proxy = nil
	u, err := url.Parse(serverURL)
	if err != nil {
		t.Fatal(err)
	}
	s.client.Transport = testTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != statusEndpoint && r.URL.String() != releaseEndpoint {
			t.Error("unexpected public endpoint")
		}
		copy := r.Clone(r.Context())
		address := *r.URL
		address.Scheme, address.Host = u.Scheme, u.Host
		copy.URL, copy.Host = &address, u.Host
		return s.transport.RoundTrip(copy)
	})
	t.Cleanup(func() { _ = s.ServiceShutdown() })
	return s
}

func TestStatusProjection(t *testing.T) {
	for _, state := range []string{"operational", "degraded", "partial_outage", "major_outage", "maintenance", "unknown", "not_monitored"} {
		t.Run(state, func(t *testing.T) {
			v := statusFixture()
			item := v["services"].([]any)[0].(map[string]any)
			item["status"], v["overallStatus"] = state, state
			v["additional"] = map[string]any{"ignored": true}
			dto, err := parseStatus(encoded(t, v))
			if err != nil || dto.Services[0].Status != state || dto.Services[0].Samples24H != 1 {
				t.Fatalf("projection: %+v, %v", dto, err)
			}
		})
	}
	for _, enabled := range []bool{false, true} {
		v := statusFixture()
		item := v["services"].([]any)[0].(map[string]any)
		item["enabled"], item["lastCheckedAt"], item["responseTimeMs"] = enabled, nil, nil
		item["id"], item["monitoringType"], item["status"] = "m7h-game", "udp", "not_monitored"
		dto, err := parseStatus(encoded(t, v))
		if err != nil || dto.Services[0].Enabled != enabled || dto.Services[0].LastCheckedAt != nil {
			t.Fatal(dto, err)
		}
	}
}

func TestStatusInvalidResponses(t *testing.T) {
	cases := map[string]func(map[string]any){
		"missing services": func(v map[string]any) { delete(v, "services") },
		"null services":    func(v map[string]any) { v["services"] = nil },
		"unknown enum":     func(v map[string]any) { v["overallStatus"] = "new_status" },
		"invalid date":     func(v map[string]any) { v["generatedAt"] = "2026-02-30T12:00:00Z" },
		"wrong type":       func(v map[string]any) { v["updatedAt"] = 1 },
		"duplicates":       func(v map[string]any) { v["services"] = append(v["services"].([]any), v["services"].([]any)[0]) },
		"missing nullable": func(v map[string]any) { delete(v["services"].([]any)[0].(map[string]any), "responseTimeMs") },
		"null bool":        func(v map[string]any) { v["services"].([]any)[0].(map[string]any)["enabled"] = nil },
		"missing samples":  func(v map[string]any) { delete(v["services"].([]any)[0].(map[string]any), "samples") },
	}
	for name, modify := range cases {
		t.Run(name, func(t *testing.T) {
			v := statusFixture()
			modify(v)
			if _, err := parseStatus(encoded(t, v)); err == nil {
				t.Fatal("accepted invalid status")
			}
		})
	}
	for _, tail := range []string{" {}", " true", " garbage"} {
		if _, err := parseStatus(append(encoded(t, statusFixture()), []byte(tail)...)); err == nil {
			t.Fatal("accepted trailing body")
		}
	}
}

func TestStatusIncidentProjection(t *testing.T) {
	v := statusFixture()
	v["incidents"] = []any{map[string]any{"status": "investigating", "impact": "major_outage", "services": []any{"docs"}, "resolvedAt": nil, "updatedAt": fixtureTime}}
	dto, err := parseStatus(encoded(t, v))
	if err != nil || !dto.Services[0].HasIncident {
		t.Fatal(dto, err)
	}
}

func TestReleaseCompatibility(t *testing.T) {
	for _, state := range []string{"not_provided", "none", "available"} {
		t.Run(state, func(t *testing.T) {
			v := releaseFixture()
			if state == "not_provided" {
				delete(v, "recommendedAsset")
			}
			if state == "none" {
				v["recommendedAsset"] = nil
			}
			dto, err := parseRelease(encoded(t, v), fixtureNow)
			if err != nil || dto.AssetState != state || dto.Latest.Tag != "First" {
				t.Fatal(dto, err)
			}
			if state != "available" && dto.RecommendedAsset != nil {
				t.Fatal("selected an asset locally")
			}
		})
	}
	v := releaseFixture()
	v["latest"], v["recommendedAsset"] = nil, nil
	if dto, err := parseRelease(encoded(t, v), fixtureNow); err != nil || dto.Latest != nil {
		t.Fatal(dto, err)
	}
	v = releaseFixture()
	v["recommendedAsset"].(map[string]any)["digest"] = nil
	if dto, err := parseRelease(encoded(t, v), fixtureNow); err != nil || dto.RecommendedAsset.Digest != nil {
		t.Fatal(dto, err)
	}
}

func TestReleaseInvalidResponses(t *testing.T) {
	cases := map[string]func(map[string]any){
		"missing latest":        func(v map[string]any) { delete(v, "latest") },
		"asset without release": func(v map[string]any) { v["latest"] = nil },
		"prerelease":            func(v map[string]any) { v["latest"].(map[string]any)["prerelease"] = true },
		"bad version":           func(v map[string]any) { v["latest"].(map[string]any)["version"] = 1 },
		"empty version":         func(v map[string]any) { v["latest"].(map[string]any)["version"] = " " },
		"invalid date":          func(v map[string]any) { v["fetchedAt"] = "not-a-date" },
		"bad digest":            func(v map[string]any) { v["recommendedAsset"].(map[string]any)["digest"] = "sha256:aa" },
		"missing digest":        func(v map[string]any) { delete(v["recommendedAsset"].(map[string]any), "digest") },
		"mismatched asset": func(v map[string]any) {
			v["recommendedAsset"] = map[string]any{"id": 3, "name": "Other.zip", "size": 10, "downloads": 0, "digest": nil, "url": "https://github.com/BassaPS/bassaPSLauncher/releases/download/First/Other.zip"}
		},
		"bad ID": func(v map[string]any) { v["latest"].(map[string]any)["id"] = -1 },
	}
	for name, modify := range cases {
		t.Run(name, func(t *testing.T) {
			v := releaseFixture()
			modify(v)
			if _, err := parseRelease(encoded(t, v), fixtureNow); err == nil {
				t.Fatal("accepted invalid release")
			}
		})
	}
	for _, age := range []time.Duration{7 * 24 * time.Hour, 7*24*time.Hour + time.Millisecond, -time.Minute - time.Millisecond} {
		if _, err := parseRelease(encoded(t, releaseFixture()), fixtureNow.Add(age)); err == nil {
			t.Fatalf("accepted age %v", age)
		}
	}
	for _, age := range []time.Duration{10*time.Minute - time.Millisecond, 10 * time.Minute, 7*24*time.Hour - time.Millisecond, -time.Minute} {
		if _, err := parseRelease(encoded(t, releaseFixture()), fixtureNow.Add(age)); err != nil {
			t.Fatal(age, err)
		}
	}
}

func TestTrustedURLs(t *testing.T) {
	good := "https://github.com/BassaPS/bassaPSLauncher/releases/tag/First"
	if !trustedURL(good, "/tag/First") {
		t.Fatal("rejected First")
	}
	for _, bad := range []string{
		"http://github.com/BassaPS/bassaPSLauncher/releases/tag/First",
		"https://github.com.evil.test/BassaPS/bassaPSLauncher/releases/tag/First",
		"https://github.com:443/BassaPS/bassaPSLauncher/releases/tag/First",
		"https://github.com:/BassaPS/bassaPSLauncher/releases/tag/First",
		"https://[github.com]/BassaPS/bassaPSLauncher/releases/tag/First",
		"https://user@github.com/BassaPS/bassaPSLauncher/releases/tag/First",
		good + "?x=1", good + "?", good + "#", good + "#x",
		"https://github.com/Other/bassaPSLauncher/releases/tag/First",
		"https://github.com/BassaPS/bassaPSLauncher/releases/tag/%2e%2e/First",
		"javascript:alert(1)", "data:text/html,test", "file:///First",
	} {
		if trustedURL(bad, "/tag/First") {
			t.Fatal("accepted URL", bad)
		}
	}
	if trustedURL("https://github.com/BassaPS/bassaPSLauncher/releases/tag/../First", "/tag/../First") {
		t.Fatal("accepted traversal tag")
	}
}

func TestHTTPFailuresAndHeaders(t *testing.T) {
	for _, code := range []int{200, 404, 429, 500, 503, 302} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("Accept") != "application/json" || r.Header.Get("User-Agent") != "BASSAPSLauncher-PublicInfo/1" {
					t.Error("invalid public headers")
				}
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				if code == 302 {
					w.Header().Set("Location", "/other")
				}
				w.WriteHeader(code)
				_, _ = w.Write(encoded(t, statusFixture()))
			}))
			defer server.Close()
			s := redirectedClient(t, server.URL)
			if s.client.Jar != nil {
				t.Fatal("public client has CookieJar")
			}
			_, err := s.GetStatus(context.Background())
			if (err == nil) != (code == 200) {
				t.Fatal(code, err)
			}
			if code == 302 && (err == nil || err.Error() != "platform: redirect") {
				t.Fatal(err)
			}
		})
	}
	for _, tc := range []struct{ name, contentType, body, want string }{
		{"html", "text/html", "private fixture body", "content_type"},
		{"malformed", "application/json", "private fixture body", "invalid_json"},
		{"too large", "application/json", strings.Repeat(" ", maxBody+1), "body_too_large"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			_, err := redirectedClient(t, server.URL).GetStatus(context.Background())
			if err == nil || err.Error() != "platform: "+tc.want {
				t.Fatal(err)
			}
		})
	}
}

func waitForWaiters(t *testing.T, s *PlatformService, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		f := s.flights[statusEndpoint]
		ready := f != nil && f.waiters == count
		s.mu.Unlock()
		if ready {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("waiters did not join")
}

func TestSharedRequestAndIndependentCancellation(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			close(started)
		}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(encoded(t, statusFixture()))
	}))
	defer server.Close()
	s := redirectedClient(t, server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	first, second := make(chan error, 1), make(chan error, 1)
	go func() { _, err := s.GetStatus(ctx); first <- err }()
	<-started
	go func() { _, err := s.GetStatus(context.Background()); second <- err }()
	waitForWaiters(t, s, 2)
	cancel()
	if err := <-first; err == nil || err.Error() != "platform: cancelled" {
		t.Fatal(err)
	}
	close(release)
	if err := <-second; err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("duplicate HTTP request", calls.Load())
	}
}

func TestTimeoutCancellationAndShutdown(t *testing.T) {
	for _, mode := range []string{"timeout", "cancel", "shutdown"} {
		t.Run(mode, func(t *testing.T) {
			started, ended := make(chan struct{}), make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				_, _ = io.WriteString(w, "{")
				w.(http.Flusher).Flush()
				close(started)
				<-r.Context().Done()
				close(ended)
			}))
			defer server.Close()
			s := redirectedClient(t, server.URL)
			if mode == "timeout" {
				s.client.Timeout = 30 * time.Millisecond
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() { _, err := s.GetStatus(ctx); result <- err }()
			<-started
			if mode == "cancel" {
				cancel()
			}
			if mode == "shutdown" {
				_ = s.ServiceShutdown()
			}
			if err := <-result; err == nil {
				t.Fatal("accepted interrupted body")
			}
			select {
			case <-ended:
			case <-time.After(time.Second):
				t.Fatal("HTTP was not cancelled")
			}
			if mode == "shutdown" {
				if _, err := s.GetStatus(context.Background()); err == nil || err.Error() != "platform: shutdown" {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestTLSAndConnectionFailures(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("untrusted TLS request reached handler") }))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	defer server.Close()
	if _, err := redirectedClient(t, server.URL).GetStatus(context.Background()); err == nil || err.Error() != "platform: tls" {
		t.Fatal(err)
	}
	plain := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	address := plain.URL
	plain.Close()
	if _, err := redirectedClient(t, address).GetStatus(context.Background()); err == nil || err.Error() != "platform: connection" {
		t.Fatal(err)
	}
}

func TestEnvironmentProxy(t *testing.T) {
	if mode := os.Getenv("PLATFORM_PROXY_TEST"); mode != "" {
		s := New()
		defer s.ServiceShutdown()
		address := "https://status.example.invalid/api"
		want := "http://127.0.0.1:18081"
		if mode == "http" {
			address, want = "http://status.example.invalid/api", "http://127.0.0.1:18080"
		}
		if mode == "bypass" {
			want = ""
		}
		r, _ := http.NewRequest(http.MethodGet, address, nil)
		proxy, err := s.transport.Proxy(r)
		if err != nil || (proxy == nil && want != "") || (proxy != nil && proxy.String() != want) {
			t.Fatal("proxy policy mismatch")
		}
		return
	}
	for _, mode := range []string{"https", "http", "bypass"} {
		binary, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(binary, "-test.run=^TestEnvironmentProxy$")
		for _, entry := range os.Environ() {
			key := strings.ToUpper(strings.SplitN(entry, "=", 2)[0])
			if key != "HTTP_PROXY" && key != "HTTPS_PROXY" && key != "NO_PROXY" {
				cmd.Env = append(cmd.Env, entry)
			}
		}
		cmd.Env = append(cmd.Env, "PLATFORM_PROXY_TEST="+mode, "HTTP_PROXY=http://127.0.0.1:18080", "HTTPS_PROXY=http://127.0.0.1:18081", "NO_PROXY=")
		if mode == "bypass" {
			cmd.Env = append(cmd.Env, "NO_PROXY=.example.invalid")
		}
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("proxy test %s failed: %v %s", mode, err, output)
		}
	}
}
