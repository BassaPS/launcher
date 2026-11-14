package newsService

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"time"

	"cyrene-launcher/internal/linkpolicy"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type cachedRaster struct {
	data    string
	expires time.Time
}

func (n *NewsService) rememberImages(source string, items []NewsItem) {
	n.init()
	n.mu.Lock()
	defer n.mu.Unlock()
	for key := range n.images {
		if strings.HasPrefix(key, source+":") {
			delete(n.images, key)
		}
	}
	for _, item := range items {
		if item.Image != "" {
			n.images[source+":"+item.ID] = item.Image
		}
	}
}

func (n *NewsService) OpenNewsURL(source, id, raw string) (bool, string) {
	target, err := linkpolicy.NewsURL(source, id, raw)
	if err != nil {
		return false, "invalid_news_link"
	}
	open := n.opener
	if open == nil {
		open = application.Get().Browser.OpenURL
	}
	if err := open(target); err != nil {
		return false, "news_link_open_failed"
	}
	return true, ""
}

// GetNewsImage accepts only the ID of an image from a validated news response.
// No arbitrary URL, file, authenticated proxy or infrastructure is exposed.
// Only bounded raster bytes reach the WebView; it never fetches the source URL.
func (n *NewsService) GetNewsImage(ctx context.Context, source, id string) (bool, string, string) {
	n.init()
	if (source != "register" && source != "official") || (source == "register" && !linkpolicy.UUID(id)) || (source == "official" && !linkpolicy.OfficialID(id)) {
		return false, "", "invalid_news_image"
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	select {
	case n.imageSlots <- struct{}{}:
	case <-ctx.Done():
		return false, "", "news_request_failed"
	}
	defer func() { <-n.imageSlots }()
	n.mu.Lock()
	raw, exists := n.images[source+":"+id]
	cached := n.imageCache[raw]
	n.mu.Unlock()
	if !exists {
		return false, "", "invalid_news_image"
	}
	if cached.data != "" && time.Now().Before(cached.expires) {
		return true, cached.data, ""
	}
	target, err := linkpolicy.ImageURL(source, raw)
	if err != nil || target == "" {
		return false, "", "invalid_news_image"
	}
	body, media, err := n.read(ctx, target, imageLimit, true)
	if err != nil {
		return false, "", err.Error()
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil || (format != "png" && format != "jpeg") || media != "image/"+format || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 8192 || cfg.Height > 8192 || int64(cfg.Width)*int64(cfg.Height) > 16*1024*1024 {
		return false, "", "invalid_news_image"
	}
	data := "data:" + media + ";base64," + base64.StdEncoding.EncodeToString(body)
	n.mu.Lock()
	// At most 8 raster results (~32MiB base64) plus 2 bounded in-flight reads.
	if len(n.imageCache) >= 8 {
		clear(n.imageCache)
	}
	n.imageCache[target] = cachedRaster{data: data, expires: time.Now().Add(5 * time.Minute)}
	n.mu.Unlock()
	return true, data, ""
}
