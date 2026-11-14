package appService

import (
	"errors"
	"testing"
)

func TestOpenURLNeverPassesUnvalidatedInput(t *testing.T) {
	calls := 0
	a := &AppService{opener: func(string) error { calls++; return errors.New("private OS detail") }}
	for _, raw := range []string{"javascript:void(0)", "file:///C:/fixture.exe", "data:text/plain,x", "fixture:run", "https://evil.invalid/"} {
		if ok, _ := a.OpenURL(raw); ok {
			t.Fatal("accepted invalid URL")
		}
	}
	if calls != 0 {
		t.Fatal("invalid URL reached OS")
	}
	if ok, msg := a.OpenURL("https://bassaps.com/"); ok || msg != "external_link_open_failed" || calls != 1 {
		t.Fatal("lost OS error")
	}
	a.opener = func(target string) error {
		calls++
		if target != "https://bassaps.com/" {
			t.Fatal(target)
		}
		return nil
	}
	if ok, _ := a.OpenURL("https://bassaps.com/"); !ok || calls != 2 {
		t.Fatal("valid link failed")
	}
}
