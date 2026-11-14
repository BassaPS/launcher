package fsService

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExistingFolderOperationUsesOnlyDirectories(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "game #100%")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	calls := 0
	f := &FSService{folderOpener: func(raw string) error {
		calls++
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "file" || u.Fragment != "" || strings.TrimPrefix(u.Path, "/") != filepath.ToSlash(dir) {
			t.Fatal(raw)
		}
		return nil
	}}
	if ok, msg := f.OpenFolder(dir); !ok || msg != "" || calls != 1 {
		t.Fatal("legitimate directory broken")
	}
	file := filepath.Join(t.TempDir(), "fixture.exe")
	os.WriteFile(file, []byte("not executable"), 0600)
	for _, path := range []string{file, filepath.Join(t.TempDir(), "missing"), "javascript:void(0)"} {
		if ok, _ := f.OpenFolder(path); ok {
			t.Fatal("non-directory opened")
		}
	}
	if calls != 1 {
		t.Fatal("non-directory reached OS")
	}
	f.folderOpener = func(string) error { return errors.New("OS detail") }
	if ok, msg := f.OpenFolder(dir); ok || msg != "folder_open_failed" {
		t.Fatal("folder error hidden")
	}
}
