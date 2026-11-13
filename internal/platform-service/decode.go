package platformService

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// Presence is tracked independently of value, including explicit nullable fields.
type required[T any] struct {
	value   T
	present bool
}

func (f *required[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return failure("invalid_response")
	}
	f.present = true
	return json.Unmarshal(data, &f.value)
}

type nullable[T any] struct {
	value   *T
	present bool
}

func (f *nullable[T]) UnmarshalJSON(data []byte) error {
	f.present = true
	return json.Unmarshal(data, &f.value)
}

func decode(data []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(out); err != nil {
		return failure("invalid_json")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return failure("invalid_json")
	}
	return nil
}

var timestampPattern = regexp.MustCompile(`^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,3})?Z$`)

func timestamp(value string) bool {
	_, err := time.Parse(time.RFC3339Nano, value)
	return timestampPattern.MatchString(value) && err == nil
}

func nullableTime(f nullable[string]) bool {
	return f.present && (f.value == nil || timestamp(*f.value))
}

func text(value string) bool {
	return strings.TrimSpace(value) != "" && len(value) <= 1024 && !strings.ContainsFunc(value, unicode.IsControl)
}

func status(value string) bool {
	switch value {
	case "operational", "degraded", "partial_outage", "major_outage", "maintenance", "unknown", "not_monitored":
		return true
	default:
		return false
	}
}

const maxSafeInteger = 9007199254740991

func identifier(f required[int64]) bool {
	return f.present && f.value >= 0 && f.value <= maxSafeInteger
}
