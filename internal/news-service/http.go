package newsService

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"sync"
	"time"

	"cyrene-launcher/pkg/constant"
)

const registerLimit = 1 << 20
const officialLimit = 512 << 10
const imageLimit = 3 << 20

type NewsService struct {
	once       sync.Once
	client     *http.Client
	opener     func(string) error
	mu         sync.Mutex
	images     map[string]string
	imageCache map[string]cachedRaster
	imageSlots chan struct{}
}

func newNewsClient(transport http.RoundTripper) *http.Client {
	if transport == nil {
		// Keep OS environment proxy/TLS verification; never modify shared transport.
		transport = http.DefaultTransport.(*http.Transport).Clone()
	}
	return &http.Client{Timeout: 12 * time.Second, Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("news_redirect_rejected") },
	}
}

func (n *NewsService) init() {
	n.once.Do(func() {
		if n.client == nil {
			n.client = newNewsClient(nil)
		}
		n.images = make(map[string]string)
		n.imageCache = make(map[string]cachedRaster)
		n.imageSlots = make(chan struct{}, 2)
	})
}

func (n *NewsService) read(ctx context.Context, target string, limit int64, image bool) ([]byte, string, error) {
	n.init()
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, "", errors.New("news_request_failed")
	}
	if image {
		req.Header.Set("Accept", "image/png, image/jpeg")
	} else {
		req.Header.Set("Accept", "application/json")
	}
	req.Header.Set("User-Agent", "CyreneLauncher/"+constant.CurrentLauncherVersion)
	resp, err := n.client.Do(req)
	if err != nil {
		return nil, "", errors.New("news_request_failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", errors.New("news_http_status")
	}
	media, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, "", errors.New("news_content_type")
	}
	if (!image && media != "application/json") || (image && media != "image/png" && media != "image/jpeg") {
		return nil, "", errors.New("news_content_type")
	}
	if charset := params["charset"]; !image && charset != "" && !strings.EqualFold(charset, "utf-8") {
		return nil, "", errors.New("news_content_type")
	}
	if resp.ContentLength > limit {
		return nil, "", errors.New("news_body_too_large")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, "", errors.New("news_body_read_failed")
	}
	if int64(len(body)) > limit {
		return nil, "", errors.New("news_body_too_large")
	}
	return body, media, nil
}
