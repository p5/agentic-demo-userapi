package webhook

import (
	"encoding/json"
	"time"
)

// WebhookEvent represents a parsed webhook event.
type WebhookEvent struct {
	ID        string
	Source    string
	EventType string
	Payload   json.RawMessage
	Headers   map[string]string
	CreatedAt time.Time
}
