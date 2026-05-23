// Package webhook provides types and parsers for handling GitHub and GitLab webhook events.
package webhook

import (
	"encoding/json"
	"errors"
	"net/http"
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

// Parser defines the interface for parsing webhook payloads from different sources.
type Parser interface {
	// Source returns the name of the webhook source (e.g., "github", "gitlab").
	Source() string
	// Parse extracts webhook event data from an HTTP request and body.
	Parse(r *http.Request, body []byte) (*WebhookEvent, error)
}

var (
	// ErrUnsupportedEvent is returned when the webhook event type is missing or not recognized.
	ErrUnsupportedEvent = errors.New("webhook: unsupported or missing event type")
	// ErrMalformedPayload is returned when the webhook payload is invalid or malformed.
	ErrMalformedPayload = errors.New("webhook: malformed payload")
)
