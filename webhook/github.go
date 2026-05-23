package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GitHubParser implements Parser for GitHub webhooks.
type GitHubParser struct{}

// NewGitHubParser creates a new GitHub webhook parser.
func NewGitHubParser() *GitHubParser {
	return &GitHubParser{}
}

// Source returns the source name for GitHub webhooks.
func (p *GitHubParser) Source() string {
	return "github"
}

// Parse reads an HTTP request and returns a WebhookEvent for GitHub.
// It extracts the event type from the X-GitHub-Event header and the delivery ID
// from the X-GitHub-Delivery header. If the delivery ID is missing, it generates
// one based on the current timestamp. The body must be valid JSON.
//
// This method does NOT verify webhook signatures — that is the caller's responsibility.
func (p *GitHubParser) Parse(r *http.Request, body []byte) (*WebhookEvent, error) {
	// Read event type from header
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		return nil, ErrUnsupportedEvent
	}

	// Read delivery ID from header, generate if empty
	deliveryID := r.Header.Get("X-GitHub-Delivery")
	if deliveryID == "" {
		deliveryID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Validate that body is valid JSON
	var tmp interface{}
	if err := json.Unmarshal(body, &tmp); err != nil {
		return nil, ErrMalformedPayload
	}

	// Build headers map
	headers := map[string]string{
		"X-GitHub-Event":      r.Header.Get("X-GitHub-Event"),
		"X-GitHub-Delivery":   r.Header.Get("X-GitHub-Delivery"),
		"X-Hub-Signature-256": r.Header.Get("X-Hub-Signature-256"),
	}

	// Create and return WebhookEvent
	event := &WebhookEvent{
		ID:        deliveryID,
		Source:    "github",
		EventType: eventType,
		Payload:   json.RawMessage(body),
		Headers:   headers,
		CreatedAt: time.Now(),
	}

	return event, nil
}
