// Package webhook provides types and interfaces for parsing webhook events
// from various sources like GitHub and GitLab.
package webhook

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// Package-level errors for webhook parsing.
var (
	// ErrUnsupportedEvent is returned when the webhook event type is not recognized or missing.
	ErrUnsupportedEvent = errors.New("webhook: unsupported event type")
	
	// ErrMalformedPayload is returned when the webhook payload is not valid JSON.
	ErrMalformedPayload = errors.New("webhook: malformed payload")
)

// WebhookEvent represents a parsed webhook event from any source.
type WebhookEvent struct {
	ID        string            `json:"id"`         // unique event ID
	Source    string            `json:"source"`     // e.g. "github", "gitlab"
	EventType string            `json:"event_type"` // e.g. "push", "merge_request"
	Payload   json.RawMessage   `json:"payload"`    // raw JSON payload body
	Headers   map[string]string `json:"headers"`    // selected headers
	CreatedAt time.Time         `json:"created_at"`
}

// Parser is the interface for source-specific webhook parsers.
type Parser interface {
	// Source returns the source name (e.g. "github").
	Source() string
	
	// Parse reads an HTTP request and returns a WebhookEvent.
	// It does NOT verify signatures — that is the caller's responsibility.
	Parse(r *http.Request, body []byte) (*WebhookEvent, error)
}
