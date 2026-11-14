package newsService

import (
	"context"
	"strings"
	"testing"
)

func TestOfficialFieldAndBodyBoundaries(t *testing.T) {
	for _, invalid := range []string{"1/011", "001/1", "02/31", "00/00", "ab/cd", "11/1 "} {
		if timestamp, _ := parseMonthDay(invalid); timestamp != 0 {
			t.Fatal("invalid official date", invalid)
		}
	}
	first := officialPost("one", "https://hoyo.link/461DNURXG")
	second := officialPost("two", "https://youtu.be/rMcFUTECbMQ")
	second["type"] = "POST_TYPE_ACTIVITY"
	third := officialPost("three", "https://www.hoyolab.com/article/47395082?utm_id=6")
	third["type"] = "POST_TYPE_INFO"
	badTitle := officialPost("over", "https://hoyo.link/461DNURXG")
	badTitle["title"] = strings.Repeat("界", 513)
	badDate := officialPost("date", "https://hoyo.link/461DNURXG")
	badDate["date"] = "02/31"
	badType := officialPost("type", "https://hoyo.link/461DNURXG")
	badType["type"] = "UNKNOWN"
	posts := []any{first, second, third, first, badTitle, badDate, badType}
	ok, items, msg := fixtureNews(officialBody(posts, []any{})).GetOfficialNews(context.Background(), "en-us")
	if !ok || len(items) != 3 || msg != "" {
		t.Fatal("normal categories/invalid item filtering")
	}
	types := map[string]bool{}
	for _, item := range items {
		types[item.Type] = true
	}
	for _, kind := range []string{"notice", "event", "info"} {
		if !types[kind] {
			t.Fatal("lost category", kind)
		}
	}
	valid := officialBody([]any{}, []any{})
	for _, length := range []int{officialLimit, officialLimit + 1} {
		body := valid + strings.Repeat(" ", length-len(valid))
		ok, _, _ := fixtureNews(body).GetOfficialNews(context.Background(), "en-us")
		if ok != (length == officialLimit) {
			t.Fatal("official decoded byte boundary")
		}
	}
	ninety := make([]any, 90)
	for i := range ninety {
		p := officialPost("id"+strings.Repeat("a", i/26)+string(rune('A'+i%26)), "https://hoyo.link/461DNURXG")
		ninety[i] = p
	}
	if ok, list, _ := fixtureNews(officialBody(ninety, make([]any, 30))).GetOfficialNews(context.Background(), "en-us"); !ok || len(list) != 90 {
		t.Fatal("valid count boundary rejected")
	}
}
