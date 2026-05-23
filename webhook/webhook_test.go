package webhook_test

import (
	"bytes"
	"errors"
	"net/http"
	"testing"

	"github.com/webhookd/webhookd/webhook"
)

// TestGitHubParser_Source verifies that GitHubParser returns "github" as its source.
func TestGitHubParser_Source(t *testing.T) {
	parser := webhook.NewGitHubParser()
	if got := parser.Source(); got != "github" {
		t.Errorf("Source() = %q, want %q", got, "github")
	}
}

// TestGitHubParser_Parse_Valid verifies that GitHubParser correctly parses
// a valid webhook request with proper headers and JSON body.
func TestGitHubParser_Parse_Valid(t *testing.T) {
	parser := webhook.NewGitHubParser()
	
	body := []byte(`{"action":"opened","number":42}`)
	req, err := http.NewRequest("POST", "http://example.com/webhook", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-GitHub-Delivery", "12345-67890")
	req.Header.Set("X-Hub-Signature-256", "sha256=abcdef")
	
	event, err := parser.Parse(req, body)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	
	if event.Source != "github" {
		t.Errorf("Source = %q, want %q", event.Source, "github")
	}
	if event.EventType != "pull_request" {
		t.Errorf("EventType = %q, want %q", event.EventType, "pull_request")
	}
	if event.ID != "12345-67890" {
		t.Errorf("ID = %q, want %q", event.ID, "12345-67890")
	}
	if string(event.Payload) != string(body) {
		t.Errorf("Payload = %q, want %q", string(event.Payload), string(body))
	}
	if event.Headers["X-GitHub-Event"] != "pull_request" {
		t.Errorf("Headers[X-GitHub-Event] = %q, want %q", event.Headers["X-GitHub-Event"], "pull_request")
	}
	if event.Headers["X-GitHub-Delivery"] != "12345-67890" {
		t.Errorf("Headers[X-GitHub-Delivery] = %q, want %q", event.Headers["X-GitHub-Delivery"], "12345-67890")
	}
	if event.Headers["X-Hub-Signature-256"] != "sha256=abcdef" {
		t.Errorf("Headers[X-Hub-Signature-256] = %q, want %q", event.Headers["X-Hub-Signature-256"], "sha256=abcdef")
	}
	if event.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero, want non-zero time")
	}
}

// TestGitHubParser_Parse_MissingEvent verifies that GitHubParser returns
// ErrUnsupportedEvent when the X-GitHub-Event header is missing.
func TestGitHubParser_Parse_MissingEvent(t *testing.T) {
	parser := webhook.NewGitHubParser()
	
	body := []byte(`{"action":"opened"}`)
	req, err := http.NewRequest("POST", "http://example.com/webhook", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	
	// No X-GitHub-Event header set
	
	event, err := parser.Parse(req, body)
	if !errors.Is(err, webhook.ErrUnsupportedEvent) {
		t.Errorf("Parse() error = %v, want %v", err, webhook.ErrUnsupportedEvent)
	}
	if event != nil {
		t.Errorf("Parse() event = %v, want nil", event)
	}
}

// TestGitHubParser_Parse_InvalidJSON verifies that GitHubParser returns
// ErrMalformedPayload when the body is not valid JSON.
func TestGitHubParser_Parse_InvalidJSON(t *testing.T) {
	parser := webhook.NewGitHubParser()
	
	body := []byte(`not valid json`)
	req, err := http.NewRequest("POST", "http://example.com/webhook", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", "12345")
	
	event, err := parser.Parse(req, body)
	if !errors.Is(err, webhook.ErrMalformedPayload) {
		t.Errorf("Parse() error = %v, want %v", err, webhook.ErrMalformedPayload)
	}
	if event != nil {
		t.Errorf("Parse() event = %v, want nil", event)
	}
}

// TestGitLabParser_Source verifies that GitLabParser returns "gitlab" as its source.
func TestGitLabParser_Source(t *testing.T) {
	parser := webhook.NewGitLabParser()
	if got := parser.Source(); got != "gitlab" {
		t.Errorf("Source() = %q, want %q", got, "gitlab")
	}
}

// TestGitLabParser_Parse_Valid verifies that GitLabParser correctly parses
// a valid webhook request with proper headers and JSON body.
func TestGitLabParser_Parse_Valid(t *testing.T) {
	parser := webhook.NewGitLabParser()
	
	body := []byte(`{"object_kind":"merge_request","project":{"id":123}}`)
	req, err := http.NewRequest("POST", "http://example.com/webhook", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	
	req.Header.Set("X-Gitlab-Event", "Merge Request Hook")
	req.Header.Set("X-Gitlab-Token", "secret-token")
	
	event, err := parser.Parse(req, body)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	
	if event.Source != "gitlab" {
		t.Errorf("Source = %q, want %q", event.Source, "gitlab")
	}
	if event.EventType != "Merge Request Hook" {
		t.Errorf("EventType = %q, want %q", event.EventType, "Merge Request Hook")
	}
	if event.ID == "" {
		t.Error("ID is empty, want non-empty generated ID")
	}
	if string(event.Payload) != string(body) {
		t.Errorf("Payload = %q, want %q", string(event.Payload), string(body))
	}
	if event.Headers["X-Gitlab-Event"] != "Merge Request Hook" {
		t.Errorf("Headers[X-Gitlab-Event] = %q, want %q", event.Headers["X-Gitlab-Event"], "Merge Request Hook")
	}
	if event.Headers["X-Gitlab-Token"] != "secret-token" {
		t.Errorf("Headers[X-Gitlab-Token] = %q, want %q", event.Headers["X-Gitlab-Token"], "secret-token")
	}
	if event.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero, want non-zero time")
	}
}

// TestGitLabParser_Parse_MissingEvent verifies that GitLabParser returns
// ErrUnsupportedEvent when the X-Gitlab-Event header is missing.
func TestGitLabParser_Parse_MissingEvent(t *testing.T) {
	parser := webhook.NewGitLabParser()
	
	body := []byte(`{"object_kind":"push"}`)
	req, err := http.NewRequest("POST", "http://example.com/webhook", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	
	// No X-Gitlab-Event header set
	
	event, err := parser.Parse(req, body)
	if !errors.Is(err, webhook.ErrUnsupportedEvent) {
		t.Errorf("Parse() error = %v, want %v", err, webhook.ErrUnsupportedEvent)
	}
	if event != nil {
		t.Errorf("Parse() event = %v, want nil", event)
	}
}
