// Package linkpolicy defines the URLs allowed at the launcher's OS boundary.
package linkpolicy

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

var ErrURL = errors.New("invalid_external_link")
var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var officialID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
var shortLink = regexp.MustCompile(`^/[A-Za-z0-9_-]{1,64}$`)
var articlePath = regexp.MustCompile(`^/article/[0-9]{1,20}$`)
var videoPath = regexp.MustCompile(`^/[A-Za-z0-9_-]{11}$`)
var digits = regexp.MustCompile(`^[0-9]{1,20}$`)
var cdnPath = regexp.MustCompile(`^/static-resource-v2/[A-Za-z0-9_./-]+\.(png|jpg|jpeg)$`)

func UUID(s string) bool       { return uuid.MatchString(s) }
func OfficialID(s string) bool { return officialID.MatchString(s) }

// ParseHTTPS rejects URL forms whose Go/Chromium/Windows interpretation may differ.
func ParseHTTPS(raw string) (*url.URL, error) {
	if len(raw) == 0 || len(raw) > 2048 || !strings.HasPrefix(raw, "https://") {
		return nil, ErrURL
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return nil, ErrURL
	}
	for _, value := range []string{raw, decoded} {
		for _, c := range value {
			if unicode.IsSpace(c) || unicode.IsControl(c) || unicode.Is(unicode.Cf, c) || c == '\\' {
				return nil, ErrURL
			}
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Opaque != "" || u.Host == "" || u.Host != u.Hostname() || u.Host != strings.ToLower(u.Host) {
		return nil, ErrURL
	}
	// No encoded separators, traversal, double encoding or alternate path spelling.
	if u.RawPath != "" || strings.Contains(u.Path, "%") || strings.Contains(u.Fragment, "%") || strings.Contains(u.Path, "//") {
		return nil, ErrURL
	}
	for _, segment := range strings.Split(u.Path, "/") {
		if segment == "." || segment == ".." {
			return nil, ErrURL
		}
	}
	if _, err := url.ParseQuery(u.RawQuery); err != nil {
		return nil, ErrURL
	}
	return u, nil
}

func NewsURL(source, id, raw string) (string, error) {
	u, err := ParseHTTPS(raw)
	if err != nil {
		return "", err
	}
	switch source {
	case "register":
		if !UUID(id) || u.Host != "register.bassaps.com" || u.Path != "/announcements" || u.RawQuery != "" || u.ForceQuery || u.Fragment != id || u.RawFragment != "" {
			return "", ErrURL
		}
		return "https://register.bassaps.com/announcements#" + id, nil
	case "official":
		if !OfficialID(id) {
			return "", ErrURL
		}
		switch u.Host {
		case "hoyo.link":
			if !shortLink.MatchString(u.Path) || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
				return "", ErrURL
			}
		case "www.hoyolab.com":
			if !articlePath.MatchString(u.Path) || u.Fragment != "" {
				return "", ErrURL
			}
			q := u.Query()
			for k, v := range q {
				if k != "utm_id" || len(v) != 1 || !digits.MatchString(v[0]) {
					return "", ErrURL
				}
			}
		case "youtu.be":
			if !videoPath.MatchString(u.Path) || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
				return "", ErrURL
			}
		default:
			return "", ErrURL
		}
		return u.String(), nil
	default:
		return "", ErrURL
	}
}

func ImageURL(source, raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	u, err := ParseHTTPS(raw)
	if err != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return "", ErrURL
	}
	switch source {
	case "register":
		if u.Host != "register.bassaps.com" || !strings.HasPrefix(u.Path, "/api/files/") || !UUID(strings.TrimPrefix(u.Path, "/api/files/")) {
			return "", ErrURL
		}
	case "official":
		if u.Host != "fastcdn.hoyoverse.com" || !cdnPath.MatchString(u.Path) {
			return "", ErrURL
		}
	default:
		return "", ErrURL
	}
	return u.String(), nil
}

// WebURL is deliberately limited to the existing non-news UI destinations.
// Local directories stay on the existing FSService/OpenLocalServerFolder paths.
func WebURL(raw string) (string, error) {
	u, err := ParseHTTPS(raw)
	if err != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return "", ErrURL
	}
	path := strings.TrimSuffix(u.Path, "/")
	allowed := map[string][]string{
		"bassaps.com": {"", "/news"}, "download.bassaps.com": {""},
		"docs.bassaps.com": {""}, "status.bassaps.com": {""},
		"register.bassaps.com":     {"", "/register", "/profile"},
		"github.com":               {"/BassaPS/bassaPSLauncher", "/BassaPS/bassaPSLauncher/releases", "/Mar7thDev/CyreneLauncher", "/hessiser/veritas/wiki"},
		"sranalysis.punklorde.org": {""}, "srtools.punklorde.org": {""}, "srtools.neonteam.dev": {""},
	}
	for _, p := range allowed[u.Host] {
		if path == p {
			return u.String(), nil
		}
	}
	return "", ErrURL
}
