package linkpolicy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const articleID = "f48c7f6b-3a9b-4fba-a7e9-66077aa5c4d8"
const registerArticle = "https://register.bassaps.com/announcements#" + articleID

func TestNewsSourcesAndURLAttacks(t *testing.T) {
	good := []struct{ source, id, url string }{
		{"register", articleID, registerArticle},
		{"official", "rAm1i7008b", "https://www.hoyolab.com/article/47395082?utm_id=6"},
		{"official", "rAm1i7008b", "https://hoyo.link/461DNURXG"},
		{"official", "rAm1i7008b", "https://youtu.be/rMcFUTECbMQ"},
	}
	for _, x := range good {
		if got, err := NewsURL(x.source, x.id, x.url); err != nil || got != x.url {
			t.Fatalf("valid source rejected: %q %v", x.url, err)
		}
	}
	bad := []string{
		"javascript:void(0)", "file:///C:/fixture.txt", "data:text/html,fixture", "http://register.bassaps.com/announcements#" + articleID,
		"ftp://fixture.invalid/", "fixture-handler:run", "//register.bassaps.com/announcements", "C:/fixture.txt", `\\fixture.invalid\share`,
		"https://u:p@register.bassaps.com/announcements#" + articleID, "https://register.bassaps.com.evil.invalid/announcements#" + articleID,
		"https://register.bassaps.com@evil.invalid/announcements#" + articleID, "https://register.bassaps.com:443/announcements#" + articleID,
		"https://register.bassaps.com/announcements%2f#" + articleID, "https://register.bassaps.com\\@evil.invalid/announcements",
		"https://register.bassaps.com/a/../announcements#" + articleID, "https://register.bassaps.com/announcements#" + articleID + "\n",
		registerArticle + "\x00", registerArticle + " ", "https://register.bassaps.com/%252fannouncements#" + articleID,
		"https://register.bassaps.com/announcements#invalid", registerArticle + "?query", "https://register.bassaps.com/announcements?x=1#" + articleID,
		"https://register.bassaps.com/announcements#%66" + articleID[1:], "https://register.bassaps.com/announcements#" + strings.Repeat("a", 2100),
		"https://evil.invalid/", "https://hoyo.link/a/../b", "https://hoyo.link/%2e%2e/b", "https://hoyo.link/x?url=https://evil.invalid",
		"https://www.hoyolab.com/article/1?redirect=https://evil.invalid", "https://youtu.be/invalid", "https://hoyo.link/ok#file",
		"https://hoyo.link/ok%0d%0a", "https://hoyo.link/ok\u200b",
	}
	for i, raw := range bad {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			for _, source := range []string{"register", "official", "forged"} {
				if _, err := NewsURL(source, articleID, raw); err == nil {
					t.Fatalf("accepted attack %q source %s", raw, source)
				}
			}
		})
	}
	if _, err := NewsURL("register", "a48c7f6b-3a9b-4fba-a7e9-66077aa5c4d8", registerArticle); err == nil {
		t.Fatal("ID mismatch accepted")
	}
	if _, err := NewsURL("official", articleID, registerArticle); err == nil {
		t.Fatal("source forgery accepted")
	}
}

func TestImageAndExistingWebRules(t *testing.T) {
	for _, source := range []string{"register", "official"} {
		if _, err := ImageURL(source, ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ImageURL("register", "https://register.bassaps.com/api/files/"+articleID); err != nil {
		t.Fatal(err)
	}
	if _, err := ImageURL("official", "https://fastcdn.hoyoverse.com/static-resource-v2/2026/11/12/cover_123.png"); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"file:///C:/x.png", "data:image/png;base64,AA==", "https://evil.invalid/x.png", "https://fastcdn.hoyoverse.com/static-resource-v2/../x.png", "https://register.bassaps.com/api/files/" + articleID + "?x=1", "https://register.bassaps.com/api/files/invalid"} {
		for _, source := range []string{"register", "official"} {
			if _, err := ImageURL(source, raw); err == nil {
				t.Fatalf("unsafe image %q", raw)
			}
		}
	}
	for _, raw := range []string{"https://bassaps.com/", "https://bassaps.com/news", "https://register.bassaps.com/register", "https://register.bassaps.com/profile", "https://download.bassaps.com/", "https://docs.bassaps.com/", "https://status.bassaps.com/", "https://github.com/BassaPS/bassaPSLauncher/releases", "https://github.com/Mar7thDev/CyreneLauncher", "https://srtools.punklorde.org/", "https://srtools.neonteam.dev", "https://sranalysis.punklorde.org/", "https://github.com/hessiser/veritas/wiki"} {
		if _, err := WebURL(raw); err != nil {
			t.Fatalf("existing URL %q: %v", raw, err)
		}
	}
	for _, raw := range []string{"file:///C:/Windows", "fixture-handler:run", "https://evil.invalid/", registerArticle, "https://github.com/unknown/unknown", "https://bassaps.com/?redirect=file"} {
		if _, err := WebURL(raw); err == nil {
			t.Fatalf("generic bypass %q", raw)
		}
	}
}

func TestRuntimeBrowserBypassBlocked(t *testing.T) {
	for _, tc := range []struct {
		query, body string
		blocked     bool
	}{
		{"?object=9&method=0", "", true}, {"?object=09&method=0", "", true},
		{"", `{"object":9,"method":0,"args":{"url":"fixture:run"}}`, true},
		{"?object=0", `{"object":9,"method":0}`, true},
		{"?object=9", `{"object":0,"method":0}`, false},
		{"?object=10&method=0", "", false}, {"?object=6&method=0", "", false},
	} {
		calls := 0
		h := RuntimeGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			got, _ := io.ReadAll(r.Body)
			if string(got) != tc.body {
				t.Fatal("changed method body")
			}
			w.WriteHeader(204)
		}))
		r := httptest.NewRequest("POST", "/wails/runtime"+tc.query, bytes.NewBufferString(tc.body))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if tc.blocked && (w.Code != 403 || calls != 0) {
			t.Fatalf("browser bypass: %d/%d", w.Code, calls)
		}
		if !tc.blocked && (w.Code != 204 || calls != 1) {
			t.Fatal("unrelated runtime blocked")
		}
	}
}
