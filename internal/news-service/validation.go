package newsService

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"
	"unicode/utf8"

	"cyrene-launcher/internal/linkpolicy"
)

// Register's title/summary limits are JavaScript UTF-16 code units, not bytes.
func textLimit(s string, limit int) bool {
	if !utf8.ValidString(s) || len(s) > limit*4 {
		return false
	}
	units := 0
	for _, r := range s {
		units++
		if r > 0xffff {
			units++
		}
		if units > limit {
			return false
		}
	}
	return true
}

func customItems(body []byte) ([]NewsItem, error) {
	trimmed := bytes.TrimSpace(body)
	if !utf8.Valid(body) || len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, errors.New("news_invalid_json")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if _, err := decoder.Token(); err != nil {
		return nil, errors.New("news_invalid_list")
	}
	records := make([]json.RawMessage, 0, 30)
	for decoder.More() {
		if len(records) >= 30 {
			return nil, errors.New("news_invalid_list")
		}
		var record json.RawMessage
		if decoder.Decode(&record) != nil {
			return nil, errors.New("news_invalid_json")
		}
		records = append(records, record)
	}
	if token, err := decoder.Token(); err != nil || token != json.Delim(']') {
		return nil, errors.New("news_invalid_json")
	}
	var extra json.RawMessage
	if decoder.Decode(&extra) != io.EOF {
		return nil, errors.New("news_invalid_json")
	}
	items := make([]NewsItem, 0, len(records))
	seen := make(map[string]bool)
	for _, raw := range records {
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) != nil {
			continue
		}
		valid := true
		for _, key := range []string{"id", "title", "intro", "image", "url", "time", "timestamp", "type", "pinned"} {
			value, ok := fields[key]
			if !ok || string(value) == "null" {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}
		var item NewsItem
		if json.Unmarshal(raw, &item) != nil || !linkpolicy.UUID(item.ID) || seen[item.ID] || !textLimit(item.Title, 100) || !textLimit(item.Intro, 240) || item.Type != "notice" || item.Timestamp < 1 || item.Timestamp > 253402300799 || len(item.Time) != 10 {
			continue
		}
		date, err := time.Parse("2006-01-02", item.Time)
		if err != nil || date.Format("2006-01-02") != time.Unix(item.Timestamp, 0).UTC().Format("2006-01-02") {
			continue
		}
		// Reject malformed API links rather than silently repairing contradictory data.
		item.URL, err = linkpolicy.NewsURL("register", item.ID, item.URL)
		if err != nil {
			continue
		}
		item.Image, _ = linkpolicy.ImageURL("register", item.Image)
		item.Title, item.Intro = stripHTML(item.Title), stripHTML(item.Intro)
		if item.Title == "" {
			continue
		}
		seen[item.ID] = true
		items = append(items, item)
	}
	return items, nil
}
