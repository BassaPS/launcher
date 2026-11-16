package platformService

import (
	"net/url"
	"regexp"
	"strings"
	"time"
)

type releaseEnvelope struct {
	Latest           nullable[releaseWire] `json:"latest"`
	RecommendedAsset nullable[assetWire]   `json:"recommendedAsset"`
	FetchedAt        required[string]      `json:"fetchedAt"`
	Stale            required[bool]        `json:"stale"`
}

type releaseWire struct {
	ID          required[int64]       `json:"id"`
	Tag         required[string]      `json:"tag"`
	Version     required[string]      `json:"version"`
	Name        required[string]      `json:"name"`
	PublishedAt required[string]      `json:"publishedAt"`
	Body        required[string]      `json:"body"`
	Prerelease  required[bool]        `json:"prerelease"`
	Draft       bool                  `json:"draft"`
	URL         required[string]      `json:"url"`
	Assets      required[[]assetWire] `json:"assets"`
}

type assetWire struct {
	ID        required[int64]  `json:"id"`
	Name      required[string] `json:"name"`
	Size      required[int64]  `json:"size"`
	URL       required[string] `json:"url"`
	Downloads required[int64]  `json:"downloads"`
	Digest    nullable[string] `json:"digest"`
}

var digestPattern = regexp.MustCompile(`(?i)^sha256:[a-f0-9]{64}$`)

func trustedURL(value, suffix string) bool {
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Host, "github.com") ||
		u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.ForceQuery || strings.Contains(value, "#") || strings.Contains(u.Path, "\\") {
		return false
	}
	matched := false
	// Keep existing public releases valid while the provider migrates repositories.
	for _, prefix := range []string{"/BassaPS/launcher/releases", "/BassaPS/bassaPSLauncher/releases"} {
		if len(u.Path) >= len(prefix) && strings.EqualFold(u.Path[:len(prefix)], prefix) && u.Path[len(prefix):] == suffix {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}
	for _, part := range strings.Split(u.Path, "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	return true
}

func parseAsset(w assetWire, tag string) (ReleaseAsset, error) {
	if !identifier(w.ID) || !w.Name.present || !text(w.Name.value) || strings.ContainsAny(w.Name.value, "/\\") ||
		!identifier(w.Size) || !identifier(w.Downloads) || !w.URL.present || !trustedURL(w.URL.value, "/download/"+tag+"/"+w.Name.value) ||
		!w.Digest.present || (w.Digest.value != nil && !digestPattern.MatchString(*w.Digest.value)) {
		return ReleaseAsset{}, failure("invalid_response")
	}
	return ReleaseAsset{ID: w.ID.value, Name: w.Name.value, Size: w.Size.value, URL: w.URL.value, Digest: w.Digest.value}, nil
}

func sameAsset(a, b ReleaseAsset) bool {
	digestEqual := a.Digest == nil && b.Digest == nil || a.Digest != nil && b.Digest != nil && *a.Digest == *b.Digest
	return a.ID == b.ID && a.Name == b.Name && a.Size == b.Size && a.URL == b.URL && digestEqual
}

func parseRelease(body []byte, now time.Time) (ReleaseSnapshot, error) {
	var wire releaseEnvelope
	if err := decode(body, &wire); err != nil {
		return ReleaseSnapshot{}, err
	}
	if !wire.Latest.present || !wire.FetchedAt.present || !timestamp(wire.FetchedAt.value) || !wire.Stale.present {
		return ReleaseSnapshot{}, failure("invalid_response")
	}
	fetched, _ := time.Parse(time.RFC3339Nano, wire.FetchedAt.value)
	if now.Sub(fetched) >= 7*24*time.Hour || fetched.Sub(now) > time.Minute {
		return ReleaseSnapshot{}, failure("expired_release")
	}
	result := ReleaseSnapshot{FetchedAt: wire.FetchedAt.value, Stale: wire.Stale.value,
		AssetState: "not_provided", DownloadURL: downloadURL}
	if wire.RecommendedAsset.present {
		result.AssetState = "none"
	}
	if wire.Latest.value == nil {
		if wire.RecommendedAsset.value != nil {
			return ReleaseSnapshot{}, failure("invalid_response")
		}
		return result, nil
	}
	w := wire.Latest.value
	if !identifier(w.ID) || !w.Tag.present || !text(w.Tag.value) || strings.Contains(w.Tag.value, "\\") ||
		!w.Version.present || !text(w.Version.value) || !w.Name.present || !text(w.Name.value) ||
		!w.PublishedAt.present || !timestamp(w.PublishedAt.value) || !w.Body.present || !w.Prerelease.present || w.Prerelease.value || w.Draft ||
		!w.URL.present || !trustedURL(w.URL.value, "/tag/"+w.Tag.value) || !w.Assets.present {
		return ReleaseSnapshot{}, failure("invalid_response")
	}
	result.Latest = &Release{ID: w.ID.value, Tag: w.Tag.value, Version: w.Version.value,
		Name: w.Name.value, PublishedAt: w.PublishedAt.value, URL: w.URL.value}
	published, _ := time.Parse(time.RFC3339Nano, w.PublishedAt.value)
	if published.Sub(now) > time.Minute {
		return ReleaseSnapshot{}, failure("invalid_response")
	}
	assets := make([]ReleaseAsset, 0, len(w.Assets.value))
	ids := make(map[int64]bool)
	for _, item := range w.Assets.value {
		asset, err := parseAsset(item, w.Tag.value)
		if err != nil || ids[asset.ID] {
			return ReleaseSnapshot{}, failure("invalid_response")
		}
		ids[asset.ID] = true
		assets = append(assets, asset)
	}
	if wire.RecommendedAsset.value != nil {
		selected, err := parseAsset(*wire.RecommendedAsset.value, w.Tag.value)
		if err != nil {
			return ReleaseSnapshot{}, err
		}
		found := false
		for _, asset := range assets {
			if sameAsset(asset, selected) {
				found = true
				break
			}
		}
		if !found {
			return ReleaseSnapshot{}, failure("invalid_response")
		}
		result.AssetState, result.RecommendedAsset = "available", &selected
	}
	return result, nil
}
