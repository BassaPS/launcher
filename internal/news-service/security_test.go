package newsService

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"hash/crc32"
	"image"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cyrene-launcher/pkg/constant"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

const fixtureID = "f48c7f6b-3a9b-4fba-a7e9-66077aa5c4d8"

func validRegister() NewsItem {
	return NewsItem{ID: fixtureID, Title: "公開記事", Intro: "概要", URL: "https://register.bassaps.com/announcements#" + fixtureID, Time: "2026-11-11", Timestamp: 1794418925, Type: "notice"}
}
func response(r *http.Request, body []byte, media string) *http.Response {
	return &http.Response{StatusCode: 200, Request: r, Header: http.Header{"Content-Type": {media}}, Body: io.NopCloser(bytes.NewReader(body)), ContentLength: -1}
}
func fixtureNews(body string) *NewsService {
	return &NewsService{client: newNewsClient(testTransport(func(r *http.Request) (*http.Response, error) {
		return response(r, []byte(body), "application/json; charset=utf-8"), nil
	}))}
}
func marshal(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRegisterValidationPreservesNormalItems(t *testing.T) {
	base := validRegister()
	for _, tc := range []struct {
		name   string
		change func(*NewsItem)
		keep   bool
	}{
		{"normal", func(*NewsItem) {}, true},
		{"html", func(i *NewsItem) { i.Title = "<b>通常</b>"; i.Intro = "<i>概要</i>" }, true},
		{"100-units", func(i *NewsItem) { i.Title = strings.Repeat("😀", 50) }, true},
		{"over-title", func(i *NewsItem) { i.Title = strings.Repeat("😀", 51) }, false},
		{"over-intro", func(i *NewsItem) { i.Intro = strings.Repeat("界", 241) }, false},
		{"invalid-id", func(i *NewsItem) { i.ID = "invalid" }, false},
		{"zero-time", func(i *NewsItem) { i.Timestamp = 0 }, false},
		{"huge-time", func(i *NewsItem) { i.Timestamp = 253402300800 }, false},
		{"bad-date", func(i *NewsItem) { i.Time = "2026-02-31" }, false},
		{"date-mismatch", func(i *NewsItem) { i.Time = "2026-11-10" }, false},
		{"wrong-fragment", func(i *NewsItem) { i.URL = "https://register.bassaps.com/announcements#other" }, false},
		{"javascript", func(i *NewsItem) { i.URL = "javascript:void(0)" }, false},
		{"file", func(i *NewsItem) { i.URL = "file:///C:/fixture.txt" }, false},
		{"data", func(i *NewsItem) { i.URL = "data:text/html,fixture" }, false},
		{"unknown-host", func(i *NewsItem) { i.URL = "https://fixture.invalid/review" }, false},
		{"unsafe-image", func(i *NewsItem) { i.Image = "javascript:void(0)" }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := base
			tc.change(&item)
			items, err := customItems([]byte(marshal(t, []NewsItem{item})))
			if err != nil || (len(items) == 1) != tc.keep {
				t.Fatalf("validation: %v %v", items, err)
			}
			if len(items) > 0 && tc.name == "unsafe-image" && items[0].Image != "" {
				t.Fatal("unsafe image retained")
			}
		})
	}
	// One bad article must not discard a valid sibling; first duplicate wins.
	bad := base
	bad.ID = "bad"
	items, err := customItems([]byte(marshal(t, []NewsItem{base, bad, base})))
	if err != nil || len(items) != 1 {
		t.Fatal("individual filtering/duplicate policy")
	}
	var fields map[string]any
	json.Unmarshal([]byte(marshal(t, base)), &fields)
	fields["futureField"] = map[string]any{"safe": true}
	if items, err := customItems([]byte(marshal(t, []any{fields}))); err != nil || len(items) != 1 {
		t.Fatal("unknown fields rejected")
	}
	for _, key := range []string{"id", "title", "intro", "url", "image", "time", "timestamp", "type", "pinned"} {
		t.Run("missing-"+key, func(t *testing.T) {
			copy := map[string]any{}
			for k, v := range fields {
				copy[k] = v
			}
			delete(copy, key)
			if items, _ := customItems([]byte(marshal(t, []any{copy}))); len(items) != 0 {
				t.Fatal("missing required field accepted")
			}
			copy[key] = nil
			if items, _ := customItems([]byte(marshal(t, []any{copy}))); len(items) != 0 {
				t.Fatal("null field accepted")
			}
		})
	}
	for _, body := range []string{"null", "{}", "[", `[] {}`, `[null]`, `[{"id":42}]`} {
		items, err := customItems([]byte(body))
		if err == nil && len(items) > 0 {
			t.Fatal("bad item accepted")
		}
		if (body == "null" || body == "{}" || body == "[" || body == `[] {}`) && err == nil {
			t.Fatal("bad response accepted")
		}
	}
	for _, size := range []int{0, 30, 31, 2001} {
		body := strings.Repeat("{},", size)
		if size > 0 {
			body = body[:len(body)-1]
		}
		_, err := customItems([]byte("[" + body + "]"))
		if (err != nil) != (size > 30) {
			t.Fatalf("wrong array limit %d", size)
		}
	}
}

func officialBody(posts any, banners any) string {
	data, _ := json.Marshal(map[string]any{"retcode": 0, "message": "OK", "data": map[string]any{"content": map[string]any{"posts": posts, "banners": banners}}})
	return string(data)
}
func officialPost(id, link string) map[string]any {
	return map[string]any{"id": id, "type": "POST_TYPE_ANNOUNCE", "title": "<b>Official news</b>", "date": "11/11", "link": link}
}
func TestOfficialContractAndIndependentRules(t *testing.T) {
	posts := []any{officialPost("rAm1i7008b", "https://www.hoyolab.com/article/47395082?utm_id=6"), officialPost("other", "https://hoyo.link/461DNURXG"), officialPost("video", "https://youtu.be/rMcFUTECbMQ")}
	banners := []any{map[string]any{"id": "banner", "image": map[string]any{"url": "https://fastcdn.hoyoverse.com/static-resource-v2/2026/11/12/cover.png", "link": "https://amzn.to/not-projected"}}}
	n := fixtureNews(officialBody(posts, banners))
	ok, items, msg := n.GetOfficialNews(context.Background(), "ja-jp")
	if !ok || msg != "" || len(items) != 3 || items[0].Title != "Official news" || items[0].Image == "" {
		t.Fatalf("normal official: %v %v %s", ok, items, msg)
	}
	for _, raw := range []string{"javascript:void(0)", "file:///C:/fixture.txt", "data:text/html,x", "https://evil.invalid/"} {
		posts = append(posts, officialPost("evil", raw))
	}
	n = fixtureNews(officialBody(posts, banners))
	if ok, items, _ := n.GetOfficialNews(context.Background(), "en-us"); !ok || len(items) != 3 {
		t.Fatal("invalid articles affected valid siblings")
	}
	for _, body := range []string{`{}`, `null`, `{"retcode":0,"data":null}`, officialBody(nil, []any{}), officialBody([]any{}, nil), officialBody(make([]any, 91), []any{}), officialBody([]any{}, make([]any, 31)), officialBody([]any{}, []any{}) + `{}`} {
		if ok, _, _ := fixtureNews(body).GetOfficialNews(context.Background(), "en-us"); ok {
			t.Fatal("bad envelope accepted")
		}
	}
	if ok, list, _ := fixtureNews(officialBody([]any{}, []any{})).GetOfficialNews(context.Background(), ""); !ok || len(list) != 0 {
		t.Fatal("valid empty official list failed")
	}
	n = &NewsService{client: newNewsClient(testTransport(func(*http.Request) (*http.Response, error) { t.Fatal("invalid language sent HTTP"); return nil, nil }))}
	if ok, _, _ := n.GetOfficialNews(context.Background(), "../../unknown"); ok {
		t.Fatal("bad language accepted")
	}
}

type countedBody struct {
	io.Reader
	read   int
	closed bool
}

func (b *countedBody) Read(p []byte) (int, error) { n, e := b.Reader.Read(p); b.read += n; return n, e }
func (b *countedBody) Close() error               { b.closed = true; return nil }
func TestHTTPMIMEBodyBoundsAndCredentials(t *testing.T) {
	for _, media := range []string{"application/json", "application/json; charset=UTF-8", "text/html", "text/plain", "application/problem+json", "application/json; charset=shift_jis", "invalid;"} {
		t.Run(media, func(t *testing.T) {
			n := &NewsService{client: newNewsClient(testTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != constant.AnnouncementUrl || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("Accept") != "application/json" {
					t.Fatal("endpoint/credentials changed")
				}
				return response(r, []byte("[]"), media), nil
			}))}
			ok, _, _ := n.GetCustomNews(context.Background())
			want := media == "application/json" || media == "application/json; charset=UTF-8"
			if ok != want {
				t.Fatal("MIME policy")
			}
			if n.client.Jar != nil || n.client.Timeout != 12*time.Second {
				t.Fatal("client contract changed")
			}
		})
	}
	for _, size := range []int{registerLimit, registerLimit + 1, 2 << 20} {
		for _, length := range []int64{-1, 1, int64(size)} {
			body := &countedBody{Reader: strings.NewReader("[]" + strings.Repeat(" ", size-2))}
			n := &NewsService{client: newNewsClient(testTransport(func(r *http.Request) (*http.Response, error) {
				x := response(r, nil, "application/json")
				x.Body = body
				x.ContentLength = length
				return x, nil
			}))}
			ok, _, _ := n.GetCustomNews(context.Background())
			if ok != (size == registerLimit) || !body.closed || body.read > registerLimit+1 {
				t.Fatalf("size/close/read limit %d %d", size, body.read)
			}
		}
	}
	body := &countedBody{Reader: strings.NewReader(strings.Repeat("x", 2<<20))}
	n := &NewsService{client: newNewsClient(testTransport(func(r *http.Request) (*http.Response, error) {
		x := response(r, nil, "text/html")
		x.StatusCode = 500
		x.Body = body
		return x, nil
	}))}
	if ok, _, msg := n.GetCustomNews(context.Background()); ok || body.read != 0 || !body.closed || strings.Contains(msg, "https:") {
		t.Fatal("HTTP error read/leak")
	}
}

func TestRedirectAndCancellation(t *testing.T) {
	for _, location := range []string{"http://fixture.invalid/downgrade", "https://fixture.invalid/other"} {
		calls := 0
		n := &NewsService{client: newNewsClient(testTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			x := response(r, nil, "application/json")
			x.StatusCode = 302
			x.Header.Set("Location", location)
			return x, nil
		}))}
		if ok, _, _ := n.GetCustomNews(context.Background()); ok || calls != 1 {
			t.Fatal("redirect followed")
		}
	}
	n := &NewsService{client: newNewsClient(testTransport(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() }))}
	n.client.Timeout = 25 * time.Millisecond
	if ok, _, _ := n.GetCustomNews(context.Background()); ok {
		t.Fatal("timeout accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if ok, _, _ := n.GetCustomNews(ctx); ok {
		t.Fatal("cancel ignored")
	}
}

func TestGzipAndChunkedDecodedSizeLimit(t *testing.T) {
	for _, encoded := range []bool{false, true} {
		for _, size := range []int{registerLimit, registerLimit + 1} {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if encoded {
					w.Header().Set("Content-Encoding", "gzip")
					z := gzip.NewWriter(w)
					io.WriteString(z, "[]"+strings.Repeat(" ", size-2))
					z.Close()
				} else {
					w.(http.Flusher).Flush()
					io.WriteString(w, "[]"+strings.Repeat(" ", size-2))
				}
			}))
			n := &NewsService{client: newNewsClient(server.Client().Transport)}
			body, _, err := n.read(context.Background(), server.URL, registerLimit, false)
			server.Close()
			if (err == nil) != (size == registerLimit) || (err == nil && len(body) != size) {
				t.Fatalf("decoded size: gzip=%v size=%d err=%v", encoded, size, err)
			}
		}
	}
}

func TestImagesAreBoundedNonRedirectingRaster(t *testing.T) {
	var pngData bytes.Buffer
	png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	calls := 0
	n := &NewsService{client: newNewsClient(testTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Fatal("image credentials")
		}
		return response(r, pngData.Bytes(), "image/png"), nil
	}))}
	item := validRegister()
	item.Image = "https://register.bassaps.com/api/files/" + fixtureID
	n.rememberImages("register", []NewsItem{item})
	for range 2 {
		ok, data, msg := n.GetNewsImage(context.Background(), "register", fixtureID)
		if !ok || msg != "" || !strings.HasPrefix(data, "data:image/png;base64,") {
			t.Fatal("normal image failed")
		}
	}
	if calls != 1 {
		t.Fatal("bounded image cache missed")
	}
	if ok, _, _ := n.GetNewsImage(context.Background(), "register", "https://evil.invalid/"); ok || calls != 1 {
		t.Fatal("arbitrary fetch input accepted")
	}
	for _, media := range []string{"image/svg+xml", "text/html", "image/jpeg"} {
		bad := &NewsService{client: newNewsClient(testTransport(func(r *http.Request) (*http.Response, error) { return response(r, pngData.Bytes(), media), nil }))}
		bad.rememberImages("register", []NewsItem{item})
		if ok, _, _ := bad.GetNewsImage(context.Background(), "register", fixtureID); ok {
			t.Fatal("invalid image accepted")
		}
	}
	redirects := 0
	bad := &NewsService{client: newNewsClient(testTransport(func(r *http.Request) (*http.Response, error) {
		redirects++
		x := response(r, nil, "image/png")
		x.StatusCode = 302
		x.Header.Set("Location", "https://evil.invalid/image.png")
		return x, nil
	}))}
	bad.rememberImages("register", []NewsItem{item})
	if ok, _, _ := bad.GetNewsImage(context.Background(), "register", fixtureID); ok || redirects != 1 {
		t.Fatal("image redirect followed")
	}
}

func TestImageSizeDimensionsAndCacheExpiry(t *testing.T) {
	var encoded bytes.Buffer
	png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	for _, tc := range []struct {
		name string
		body []byte
	}{
		{"over-bytes", bytes.Repeat([]byte{'x'}, imageLimit+1)},
		{"invalid-raster", []byte("not a PNG")},
		{"over-dimensions", func() []byte {
			b := bytes.Clone(encoded.Bytes())
			binary.BigEndian.PutUint32(b[16:20], 8193)
			binary.BigEndian.PutUint32(b[29:33], crc32.ChecksumIEEE(b[12:29]))
			return b
		}()},
		{"over-pixels", func() []byte {
			b := bytes.Clone(encoded.Bytes())
			binary.BigEndian.PutUint32(b[16:20], 8192)
			binary.BigEndian.PutUint32(b[20:24], 8192)
			binary.BigEndian.PutUint32(b[29:33], crc32.ChecksumIEEE(b[12:29]))
			return b
		}()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := &NewsService{client: newNewsClient(testTransport(func(r *http.Request) (*http.Response, error) { return response(r, tc.body, "image/png"), nil }))}
			item := validRegister()
			item.Image = "https://register.bassaps.com/api/files/" + fixtureID
			n.rememberImages("register", []NewsItem{item})
			if ok, _, _ := n.GetNewsImage(context.Background(), "register", fixtureID); ok {
				t.Fatal("unbounded image accepted")
			}
		})
	}
	calls := 0
	n := &NewsService{client: newNewsClient(testTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return response(r, encoded.Bytes(), "image/png"), nil
	}))}
	item := validRegister()
	item.Image = "https://register.bassaps.com/api/files/" + fixtureID
	n.rememberImages("register", []NewsItem{item})
	if ok, _, _ := n.GetNewsImage(context.Background(), "register", fixtureID); !ok {
		t.Fatal("normal cache fill")
	}
	n.mu.Lock()
	old := n.imageCache[item.Image]
	old.expires = time.Now().Add(-time.Second)
	n.imageCache[item.Image] = old
	n.mu.Unlock()
	if ok, _, _ := n.GetNewsImage(context.Background(), "register", fixtureID); !ok || calls != 2 {
		t.Fatal("expired raster not reloaded")
	}
	for index := 0; index < 9; index++ {
		i := validRegister()
		i.ID = fixtureID[:35] + string(rune('0'+index))
		i.URL = "https://register.bassaps.com/announcements#" + i.ID
		i.Image = "https://register.bassaps.com/api/files/" + i.ID
		n.rememberImages("register", []NewsItem{i})
		if ok, _, _ := n.GetNewsImage(context.Background(), "register", i.ID); !ok {
			t.Fatal("cache fixture failed")
		}
		if len(n.imageCache) > 8 {
			t.Fatal("unbounded cache")
		}
	}
}

func TestNewsOSBoundaryAndWailsRegistry(t *testing.T) {
	calls := 0
	n := &NewsService{opener: func(string) error { calls++; return errors.New("private OS failure") }}
	for _, raw := range []string{"javascript:void(0)", "file:///C:/fixture.txt", "data:text/plain,x", "https://evil.invalid/"} {
		if ok, _ := n.OpenNewsURL("register", fixtureID, raw); ok {
			t.Fatal("invalid link accepted")
		}
	}
	if calls != 0 {
		t.Fatal("unsafe URL reached OS")
	}
	if ok, msg := n.OpenNewsURL("register", fixtureID, validRegister().URL); ok || msg != "news_link_open_failed" || calls != 1 {
		t.Fatal("OS error hidden")
	}
	n.opener = func(target string) error {
		calls++
		if target != validRegister().URL {
			t.Fatal(target)
		}
		return nil
	}
	if ok, _ := n.OpenNewsURL("register", fixtureID, validRegister().URL); !ok || calls != 2 {
		t.Fatal("normal opener call")
	}
	application.New(application.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	bindings := application.NewBindings(nil, nil)
	if err := bindings.Add(application.NewService(n)); err != nil {
		t.Fatal(err)
	}
	method := bindings.Get(&application.CallOptions{MethodName: "cyrene-launcher/internal/news-service.NewsService.OpenNewsURL"})
	if method == nil || bindings.GetByID(method.ID) != method {
		t.Fatal("missing runtime method")
	}
	args := []json.RawMessage{json.RawMessage(`"register"`), json.RawMessage(marshal(t, fixtureID)), json.RawMessage(`"javascript:void(0)"`)}
	if _, err := method.Call(context.Background(), args); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal("direct Wails dispatch bypassed validation")
	}
}
