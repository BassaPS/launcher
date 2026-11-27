//go:build windows

package march7thhoneyService

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDLLRefreshesCachedHook(t *testing.T) {
	current := []byte("current hook")
	for _, tt := range []struct {
		name   string
		cached []byte
	}{
		{"missing", nil},
		{"old size", []byte("old")},
		{"same size stale", []byte("outdated dll")},
		{"current", current},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "patch", "CyreneHook.dll")
			if tt.cached != nil {
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, tt.cached, 0644); err != nil {
					t.Fatal(err)
				}
			}
			service := New(current, nil)
			got, err := service.ensureDLLAt(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != path {
				t.Fatalf("path = %q, want %q", got, path)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(data, current) {
				t.Fatalf("cached DLL was not refreshed: %q", data)
			}
		})
	}
}

func TestEnsureDLLRejectsMissingEmbeddedHook(t *testing.T) {
	path := filepath.Join(t.TempDir(), "CyreneHook.dll")
	if err := os.WriteFile(path, []byte("cached"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := New(nil, nil).ensureDLLAt(path); err == nil {
		t.Fatal("expected error for missing embedded DLL")
	}
}
