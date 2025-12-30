package logging

import (
	"encoding/json"
	"time"
)

// LogEntry represents a single structured log entry.
type LogEntry struct {
	Time      time.Time              `json:"time"`
	Level     string                 `json:"level"`
	Message   string                 `json:"msg"`
	RequestID string                 `json:"request_id,omitempty"`
	Attrs     map[string]interface{} `json:"attrs,omitempty"`
}

// ToJSON serializes the entry for SSE transmission.
func (e *LogEntry) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}
