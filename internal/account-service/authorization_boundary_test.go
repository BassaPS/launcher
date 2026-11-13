package accountService

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestAuthorizeAdminSessionBoundary(t *testing.T) {
	for _, tc := range []struct {
		name    string
		token   string
		status  int
		body    string
		allowed bool
	}{
		{"anonymous", "", 200, `{"user":{"role":"admin"}}`, false},
		{"regular user", "fixture-token", 200, `{"user":{"role":"user"}}`, false},
		{"expired token", "fixture-token", 401, `{"user":{"role":"admin"}}`, false},
		{"forbidden", "fixture-token", 403, `{"user":{"role":"admin"}}`, false},
		{"server error", "fixture-token", 500, `{"user":{"role":"admin"}}`, false},
		{"invalid JSON", "fixture-token", 200, `invalid`, false},
		{"missing role", "fixture-token", 200, `{"user":{}}`, false},
		{"administrator", "fixture-token", 200, `{"user":{"role":"admin"}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != "/api/launcher/me" || r.Header.Get("Authorization") != "Bearer fixture-token" {
					t.Error("unexpected authorization request")
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			service := New(server.URL)
			service.token = tc.token
			cached := &Profile{Role: "admin"}
			service.profile = cached
			if err := service.AuthorizeAdmin(); (err == nil) != tc.allowed {
				t.Fatalf("AuthorizeAdmin() = %v, allowed = %v", err, tc.allowed)
			}
			if service.token != tc.token || service.profile != cached || service.profile.Role != "admin" {
				t.Fatal("authorization changed the saved session or cached role")
			}
			wantCalls := int32(1)
			if tc.token == "" {
				wantCalls = 0
			}
			if calls.Load() != wantCalls {
				t.Fatalf("authorization requests = %d, want %d", calls.Load(), wantCalls)
			}
		})
	}
}

type authorizationFailureTransport struct{ err error }

func (transport authorizationFailureTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, transport.err
}

func TestAuthorizeAdminTransportFailure(t *testing.T) {
	for _, failure := range []error{context.DeadlineExceeded, errors.New("fixture connection failure")} {
		service := New("http://127.0.0.1")
		service.token = "fixture-token"
		service.profile = &Profile{Role: "admin"}
		service.client.Transport = authorizationFailureTransport{err: failure}
		if err := service.AuthorizeAdmin(); err == nil {
			t.Fatalf("authorization accepted transport failure %v", failure)
		}
	}
}

func TestAuthorizeAdminWailsDispatch(t *testing.T) {
	// Construct only the binding registry. App.Run and AccountService startup are
	// deliberately absent, so this test never opens a WebView or loads a token.
	application.New(application.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	var role atomic.Value
	role.Store("user")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"user":{"role":"`+role.Load().(string)+`"}}`)
	}))
	defer server.Close()
	service := New(server.URL)
	bindings := application.NewBindings(nil, nil)
	if err := bindings.Add(application.NewService(service)); err != nil {
		t.Fatal(err)
	}
	method := bindings.Get(&application.CallOptions{
		MethodName: "cyrene-launcher/internal/account-service.AccountService.AuthorizeAdmin",
	})
	if method == nil || bindings.GetByID(method.ID) != method {
		t.Fatal("AuthorizeAdmin is absent from the runtime dispatch registry")
	}
	if _, err := method.Call(context.Background(), nil); err == nil {
		t.Fatal("direct Wails dispatch authorized an anonymous session")
	}
	t.Logf("registered AuthorizeAdmin method ID = %d", method.ID)
	service.token = "fixture-token"
	if _, err := method.Call(context.Background(), nil); err == nil {
		t.Fatal("direct Wails dispatch authorized a regular user")
	}
	role.Store("admin")
	if _, err := method.Call(context.Background(), nil); err != nil {
		t.Fatalf("direct Wails dispatch rejected an administrator: %v", err)
	}
	role.Store("user")
	if _, err := method.Call(context.Background(), nil); err == nil {
		t.Fatal("direct Wails dispatch retained privileges after demotion")
	}
}
