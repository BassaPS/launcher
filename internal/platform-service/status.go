package platformService

type statusWire struct {
	OverallStatus required[string]         `json:"overallStatus"`
	UpdatedAt     nullable[string]         `json:"updatedAt"`
	GeneratedAt   required[string]         `json:"generatedAt"`
	Services      required[[]serviceWire]  `json:"services"`
	Incidents     required[[]incidentWire] `json:"incidents"`
}

type serviceWire struct {
	ID             required[string]                     `json:"id"`
	Name           required[string]                     `json:"name"`
	MonitoringType required[string]                     `json:"monitoringType"`
	Enabled        required[bool]                       `json:"enabled"`
	Status         required[string]                     `json:"status"`
	LastCheckedAt  nullable[string]                     `json:"lastCheckedAt"`
	ResponseTimeMs nullable[float64]                    `json:"responseTimeMs"`
	Samples        required[map[string]required[int64]] `json:"samples"`
}

type incidentWire struct {
	Status     required[string]   `json:"status"`
	Impact     required[string]   `json:"impact"`
	Services   required[[]string] `json:"services"`
	ResolvedAt nullable[string]   `json:"resolvedAt"`
	UpdatedAt  required[string]   `json:"updatedAt"`
}

func parseStatus(body []byte) (StatusSnapshot, error) {
	var wire statusWire
	if err := decode(body, &wire); err != nil {
		return StatusSnapshot{}, err
	}
	if !wire.OverallStatus.present || !status(wire.OverallStatus.value) || !nullableTime(wire.UpdatedAt) ||
		!wire.GeneratedAt.present || !timestamp(wire.GeneratedAt.value) || !wire.Services.present || !wire.Incidents.present {
		return StatusSnapshot{}, failure("invalid_response")
	}
	incidentServices := make(map[string]bool)
	for _, incident := range wire.Incidents.value {
		if !incident.Status.present || !incident.Impact.present || !status(incident.Impact.value) ||
			!incident.Services.present || !nullableTime(incident.ResolvedAt) || !incident.UpdatedAt.present || !timestamp(incident.UpdatedAt.value) {
			return StatusSnapshot{}, failure("invalid_response")
		}
		switch incident.Status.value {
		case "investigating", "identified", "monitoring", "resolved":
		default:
			return StatusSnapshot{}, failure("invalid_response")
		}
		for _, id := range incident.Services.value {
			if !text(id) {
				return StatusSnapshot{}, failure("invalid_response")
			}
			if incident.ResolvedAt.value == nil && incident.Status.value != "resolved" {
				incidentServices[id] = true
			}
		}
	}
	result := StatusSnapshot{OverallStatus: wire.OverallStatus.value, UpdatedAt: wire.UpdatedAt.value,
		GeneratedAt: wire.GeneratedAt.value, Services: make([]StatusService, 0, len(wire.Services.value))}
	ids := make(map[string]bool)
	for _, item := range wire.Services.value {
		samples := item.Samples.value["24h"]
		if !item.ID.present || !text(item.ID.value) || ids[item.ID.value] || !item.Name.present || !text(item.Name.value) ||
			!item.MonitoringType.present || !text(item.MonitoringType.value) || !item.Enabled.present || !item.Status.present || !status(item.Status.value) ||
			!nullableTime(item.LastCheckedAt) || !item.ResponseTimeMs.present || (item.ResponseTimeMs.value != nil && *item.ResponseTimeMs.value < 0) ||
			!item.Samples.present || !identifier(samples) {
			return StatusSnapshot{}, failure("invalid_response")
		}
		ids[item.ID.value] = true
		result.Services = append(result.Services, StatusService{ID: item.ID.value, Name: item.Name.value,
			MonitoringType: item.MonitoringType.value, Enabled: item.Enabled.value, Status: item.Status.value,
			LastCheckedAt: item.LastCheckedAt.value, ResponseTimeMs: item.ResponseTimeMs.value,
			Samples24H: samples.value, HasIncident: incidentServices[item.ID.value]})
	}
	return result, nil
}
