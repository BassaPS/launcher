package newsService

import (
	"context"
	"cyrene-launcher/internal/linkpolicy"
	"cyrene-launcher/pkg/constant"
	"encoding/json"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// NewsItem is the unified shape returned for all news sources.
type NewsItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Intro     string `json:"intro"`
	Image     string `json:"image"`
	URL       string `json:"url"`
	Time      string `json:"time"`      // formatted YYYY-MM-DD
	Timestamp int64  `json:"timestamp"` // unix seconds, for sorting
	Type      string `json:"type"`      // "notice" / "event" / "info" / "" (server)
	Pinned    bool   `json:"pinned"`    // server announcements only
}

// ───────── shared helpers ─────────

var (
	htmlTagRe    = regexp.MustCompile(`<[^>]*>`)
	whitespaceRe = regexp.MustCompile(`\s+`)
)

func stripHTML(s string) string {
	if s == "" {
		return ""
	}
	s = strings.NewReplacer(
		"<br>", " ", "<br/>", " ", "<br />", " ",
		"</p>", " ", "</div>", " ", "</li>", " ",
	).Replace(s)
	s = htmlTagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = whitespaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// ───────── HoYoPlay (HYP) Game Content API ─────────

type hypContentResp struct {
	Retcode *int   `json:"retcode"`
	Message string `json:"message"`
	Data    *struct {
		Content *struct {
			Banners *[]struct {
				ID    string `json:"id"`
				Image struct {
					URL  string `json:"url"`
					Link string `json:"link"`
				} `json:"image"`
			} `json:"banners"`
			Posts *[]struct {
				ID    string `json:"id"`
				Type  string `json:"type"`
				Title string `json:"title"`
				Link  string `json:"link"`
				Date  string `json:"date"` // "MM/DD" — year is implicit
			} `json:"posts"`
		} `json:"content"`
	} `json:"data"`
}

// Map HYP POST_TYPE_X → our internal short type.
var postTypeMap = map[string]string{
	"POST_TYPE_ANNOUNCE": "notice",
	"POST_TYPE_ACTIVITY": "event",
	"POST_TYPE_INFO":     "info",
}

// parseMonthDay turns a "MM/DD" string into a unix timestamp + ISO date.
// HYP drops the year, so we infer: assume current year, but if the resulting
// date is more than a month in the future, treat it as last year (e.g. "12/30"
// returned in January).
func parseMonthDay(s string) (int64, string) {
	if len(s) != 5 || s[2] != '/' || s[0] < '0' || s[0] > '9' || s[1] < '0' || s[1] > '9' || s[3] < '0' || s[3] > '9' || s[4] < '0' || s[4] > '9' {
		return 0, ""
	}
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, s
	}
	month, err1 := strconv.Atoi(parts[0])
	day, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || month < 1 || month > 12 || day < 1 || day > 31 {
		return 0, s
	}
	now := time.Now()
	year := now.Year()
	candidate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if candidate.Month() != time.Month(month) || candidate.Day() != day {
		return 0, ""
	}
	if candidate.Sub(now) > 31*24*time.Hour {
		candidate = candidate.AddDate(-1, 0, 0)
	}
	return candidate.Unix(), candidate.Format("2006-01-02")
}

// GetOfficialNews fetches the HoYoPlay content for HSR (banners + 3-type posts)
// in a single HTTP request. Frontend filters posts by type for each tab.
func (n *NewsService) GetOfficialNews(ctx context.Context, lang string) (bool, []NewsItem, string) {
	if lang == "" {
		lang = "en-us"
	}

	allowedLanguage := map[string]bool{"en-us": true, "zh-cn": true, "zh-tw": true, "ja-jp": true, "ko-kr": true, "ru-ru": true, "vi-vn": true, "th-th": true, "id-id": true, "de-de": true, "fr-fr": true, "pt-pt": true, "es-es": true}
	if !allowedLanguage[lang] {
		return false, nil, "news_invalid_language"
	}
	q := url.Values{}
	q.Set("launcher_id", constant.HSRLauncherID)
	q.Set("game_id", constant.HSRGameID)
	q.Set("language", lang)
	apiURL := constant.HSRLauncherContentURL + "?" + q.Encode()

	body, _, err := n.read(ctx, apiURL, officialLimit, false)
	if err != nil {
		return false, nil, err.Error()
	}

	var result hypContentResp
	if err := json.Unmarshal(body, &result); err != nil || !utf8.Valid(body) {
		return false, nil, "news_invalid_json"
	}
	if result.Retcode == nil || *result.Retcode != 0 || result.Data == nil || result.Data.Content == nil || result.Data.Content.Posts == nil || result.Data.Content.Banners == nil || len(*result.Data.Content.Posts) > 90 || len(*result.Data.Content.Banners) > 30 {
		return false, nil, "news_invalid_envelope"
	}

	// Use first banner image as shared cover fallback — posts have no per-item image.
	bannerImg := ""
	if len(*result.Data.Content.Banners) > 0 {
		bannerImg, _ = linkpolicy.ImageURL("official", (*result.Data.Content.Banners)[0].Image.URL)
	}

	items := make([]NewsItem, 0, len(*result.Data.Content.Posts))
	seen := make(map[string]bool)
	for _, p := range *result.Data.Content.Posts {
		if !linkpolicy.OfficialID(p.ID) || seen[p.ID] || !textLimit(p.Title, 512) || stripHTML(p.Title) == "" {
			continue
		}
		safeLink, err := linkpolicy.NewsURL("official", p.ID, p.Link)
		if err != nil {
			continue
		}
		ourType, ok := postTypeMap[p.Type]
		if !ok {
			continue
		}
		ts, isoDate := parseMonthDay(p.Date)
		if ts == 0 || len(p.Date) != 5 {
			continue
		}
		seen[p.ID] = true
		items = append(items, NewsItem{
			ID:        p.ID,
			Title:     stripHTML(p.Title),
			Intro:     "",
			Image:     bannerImg,
			URL:       safeLink,
			Time:      isoDate,
			Timestamp: ts,
			Type:      ourType,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp > items[j].Timestamp
	})

	n.rememberImages("official", items)
	return true, items, ""
}

// ───────── Custom / Server announcements ─────────

// GetCustomNews fetches server announcements from the Cyrene website API
// (constant.AnnouncementUrl), which returns a plain []NewsItem array with
// pinned posts first.
func (n *NewsService) GetCustomNews(ctx context.Context) (bool, []NewsItem, string) {
	src := constant.AnnouncementUrl
	if src == "" {
		return false, nil, "ANNOUNCEMENT_URL_NOT_CONFIGURED"
	}

	body, _, err := n.read(ctx, src, registerLimit, false)
	if err != nil {
		return false, nil, err.Error()
	}

	items, err := customItems(body)
	if err != nil {
		return false, nil, err.Error()
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Pinned != items[j].Pinned {
			return items[i].Pinned
		}
		return items[i].Timestamp > items[j].Timestamp
	})
	n.rememberImages("register", items)
	return true, items, ""
}
