package platformService

// The platform contract snapshot uses revision 2026-11-13.2A.1.
// DTOs contain display information only, without account or update state.
type StatusService struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	MonitoringType string   `json:"monitoringType"`
	Enabled        bool     `json:"enabled"`
	Status         string   `json:"status"`
	LastCheckedAt  *string  `json:"lastCheckedAt"`
	ResponseTimeMs *float64 `json:"responseTimeMs"`
	Samples24H     int64    `json:"samples24H"`
	HasIncident    bool     `json:"hasIncident"`
}

type StatusSnapshot struct {
	OverallStatus string          `json:"overallStatus"`
	UpdatedAt     *string         `json:"updatedAt"`
	GeneratedAt   string          `json:"generatedAt"`
	Services      []StatusService `json:"services"`
}

type ReleaseAsset struct {
	ID     int64   `json:"id"`
	Name   string  `json:"name"`
	Size   int64   `json:"size"`
	URL    string  `json:"url"`
	Digest *string `json:"digest"`
}

type Release struct {
	ID          int64  `json:"id"`
	Tag         string `json:"tag"`
	Version     string `json:"version"`
	Name        string `json:"name"`
	PublishedAt string `json:"publishedAt"`
	URL         string `json:"url"`
}

type ReleaseSnapshot struct {
	Latest           *Release      `json:"latest"`
	RecommendedAsset *ReleaseAsset `json:"recommendedAsset"`
	// not_provided is the legacy envelope; none is an explicit null.
	AssetState  string `json:"assetState"`
	FetchedAt   string `json:"fetchedAt"`
	Stale       bool   `json:"stale"`
	DownloadURL string `json:"downloadUrl"`
}
