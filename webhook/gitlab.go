package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GitLabParser implements Parser for GitLab webhooks.
type GitLabParser struct{}

// NewGitLabParser creates a new GitLab webhook parser.
func NewGitLabParser() *GitLabParser {
	return &GitLabParser{}
}

// Source returns the source name for GitLab webhooks.
func (p *GitLabParser) Source() string {
	return "gitlab"
}

// Parse reads an HTTP request and returns a WebhookEvent for GitLab.
// It extracts the event type from the X-Gitlab-Event header and generates
// a unique ID based on the current timestamp. The body must be valid JSON.
//
// This method does NOT verify webhook tokens — that is the caller's responsibility.
func (p *GitLabParser) Parse(r *http.Request, body []byte) (*WebhookEvent, error) {
	// Read event type from header
	eventType := r.Header.Get("X-Gitlab-Event")
	if eventType == "" {
		return nil, ErrUnsupportedEvent
	}

	// Generate ID
	eventID := fmt.Sprintf("%d", time.Now().UnixNano())

	// Validate that body is valid JSON
	var tmp interface{}
	if err := json.Unmarshal(body, &tmp); err != nil {
		return nil, ErrMalformedPayload
	}

	// Build headers map
	headers := map[string]string{
		"X-Gitlab-Event": r.Header.Get("X-Gitlab-Event"),
		"X-Gitlab-Token": r.Header.Get("X-Gitlab-Token"),
	}

	// Create and return WebhookEvent
	event := &WebhookEvent{
		ID:        eventID,
		Source:    "gitlab",
		EventType: eventType,
		Payload:   json.RawMessage(body),
		Headers:   headers,
		CreatedAt: time.Now(),
	}

	return event, nil
}
