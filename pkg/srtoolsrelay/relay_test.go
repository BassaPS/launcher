package srtoolsrelay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRelayBoundaryAndValidation(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/sr-tools" {
			t.Errorf("path = %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"username":"123456","password":"test"}` {
			t.Errorf("unexpected body")
		}
		w.WriteHeader(400)
		_, _ = w.Write([]byte("UID validation"))
	}))
	defer upstream.Close()
	relay, err := Start(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer relay.Close()
	cases := []struct {
		name, origin, host, path, method, body string
		want                                   int
	}{
		{"forward", Origin, "", "/sr-tools", "POST", `{"username":"123456","password":"test"}`, 400},
		{"foreign origin", "https://evil.example", "", "/sr-tools", "POST", "{}", 403},
		{"missing origin", "", "", "/sr-tools", "POST", "{}", 403},
		{"dns rebinding", Origin, "evil.example:24680", "/sr-tools", "POST", "{}", 403},
		{"admin route", Origin, "", "/admin", "POST", "{}", 404},
		{"query", Origin, "", "/sr-tools?url=https://evil.example", "POST", "{}", 404},
		{"method", Origin, "", "/sr-tools", "GET", "", 405},
		{"oversized", Origin, "", "/sr-tools", "POST", strings.Repeat("x", maxRequest+1), 413},
		{"preflight", Origin, "", "/sr-tools", "OPTIONS", "", 204},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req, _ := http.NewRequest(c.method, relay.URL()+c.path, strings.NewReader(c.body))
			req.Header.Set("Origin", c.origin)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Access-Control-Request-Private-Network", "true")
			if c.host != "" {
				req.Host = c.host
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != c.want {
				t.Fatalf("status %d want %d", res.StatusCode, c.want)
			}
			if c.name == "forward" {
				b, _ := io.ReadAll(res.Body)
				if string(b) != "UID validation" {
					t.Fatal("upstream error was hidden")
				}
			}
			if c.name == "preflight" && res.Header.Get("Access-Control-Allow-Private-Network") != "true" {
				t.Fatal("missing private network permission")
			}
		})
	}
	if calls.Load() != 1 {
		t.Fatalf("unauthorized upstream calls: %d", calls.Load())
	}
}

func TestInvalidTargetsAndRedirect(t *testing.T) {
	for _, target := range []string{"", "ftp://example.com", "https://user:pass@example.com", "http://example.com?target=x"} {
		if r, err := Start(target); err == nil {
			r.Close()
			t.Fatalf("accepted %q", target)
		}
	}
	var redirected atomic.Bool
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected.Store(true) }))
	defer destination.Close()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, 307) }))
	defer upstream.Close()
	relay, err := Start(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer relay.Close()
	req, _ := http.NewRequest("POST", relay.URL()+"/sr-tools", strings.NewReader("{}"))
	req.Header.Set("Origin", Origin)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 502 || redirected.Load() {
		t.Fatal("redirect must not forward credentials")
	}
}
