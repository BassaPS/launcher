package platformService

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestReleaseEndpointAndBodyLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apps/launcher/latest" {
			t.Error("incorrect release path")
		}
		w.Header().Set("Content-Type", "application/json")
		body := encoded(t, releaseFixture())
		body = append(body, make([]byte, maxBody-len(body))...)
		for index := len(encoded(t, releaseFixture())); index < len(body); index++ {
			body[index] = ' '
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()
	dto, err := redirectedClient(t, server.URL).GetLatestRelease(context.Background())
	if err != nil || dto.Latest == nil || dto.AssetState != "available" || dto.DownloadURL != downloadURL {
		t.Fatal(dto, err)
	}
}

func TestLastWaiterCancellationAllowsRetry(t *testing.T) {
	started, stopped := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			close(started)
			<-r.Context().Done()
			close(stopped)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(encoded(t, statusFixture()))
	}))
	defer server.Close()
	s := redirectedClient(t, server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { _, err := s.GetStatus(ctx); result <- err }()
	<-started
	cancel()
	if err := <-result; err == nil {
		t.Fatal("cancelled call succeeded")
	}
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("last waiter did not cancel HTTP")
	}
	if _, err := s.GetStatus(context.Background()); err != nil {
		t.Fatal("retry failed", err)
	}
}

func TestShutdownCancelsBothEndpoints(t *testing.T) {
	started := make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-r.Context().Done()
	}))
	defer server.Close()
	s := redirectedClient(t, server.URL)
	result := make(chan error, 2)
	go func() { _, err := s.GetStatus(context.Background()); result <- err }()
	go func() { _, err := s.GetLatestRelease(context.Background()); result <- err }()
	<-started
	<-started
	if err := s.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := <-result; err == nil {
			t.Fatal("shutdown call succeeded")
		}
	}
}

func TestWailsCallContextAndPublicMethods(t *testing.T) {
	application.New(application.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	s := New()
	defer s.ServiceShutdown()
	bindings := application.NewBindings(nil, nil)
	if err := bindings.Add(application.NewService(s)); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, name := range []string{"GetStatus", "GetLatestRelease"} {
		bound := bindings.Get(&application.CallOptions{MethodName: "cyrene-launcher/internal/platform-service.PlatformService." + name})
		if bound == nil || len(bound.Inputs) != 1 {
			t.Fatal("missing context-bound method", name)
		}
		if _, err := bound.Call(ctx, nil); err == nil {
			t.Fatal("call context was not supplied")
		}
	}
	if bound := bindings.Get(&application.CallOptions{MethodName: "cyrene-launcher/internal/platform-service.PlatformService.ServiceShutdown"}); bound != nil {
		t.Fatal("lifecycle is exposed")
	}
}
