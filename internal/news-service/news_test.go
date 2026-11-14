package newsService

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCustomNewsUsesPortalAndOrdersPinnedPosts(t *testing.T) {
	first := validRegister()
	first.Title = "Update"
	first.Timestamp = 1794418926
	pinned := validRegister()
	pinned.ID = "a48c7f6b-3a9b-4fba-a7e9-66077aa5c4d8"
	pinned.URL = "https://register.bassaps.com/announcements#" + pinned.ID
	pinned.Pinned = true
	pinned.Intro = "Server maintenance"
	body, _ := json.Marshal([]NewsItem{first, pinned})
	n := fixtureNews(string(body))
	ok, items, msg := n.GetCustomNews(context.Background())
	if !ok || msg != "" || len(items) != 2 || items[0].ID != pinned.ID || items[0].Intro != "Server maintenance" || items[1].ID != first.ID {
		t.Fatalf("normal news ordering/content failed: %v %v %s", ok, items, msg)
	}
}
func TestCustomNewsRejectsInvalidResponse(t *testing.T) {
	for _, body := range []string{"<html>error</html>", "{\"items\":[]}"} {
		if ok, _, _ := fixtureNews(body).GetCustomNews(context.Background()); ok {
			t.Fatal("accepted invalid response")
		}
	}
}
