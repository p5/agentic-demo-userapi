package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/webhookd/webhookd/webhook"
)

// mockHandler is a test handler that records events it processes.
type mockHandler struct {
	mu         sync.Mutex
	events     []*webhook.WebhookEvent
	eventTypes []string
	err        error
}

func (h *mockHandler) EventTypes() []string {
	return h.eventTypes
}

func (h *mockHandler) Handle(ctx context.Context, event *webhook.WebhookEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, event)
	return h.err
}

func (h *mockHandler) getEvents() []*webhook.WebhookEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]*webhook.WebhookEvent, len(h.events))
	copy(result, h.events)
	return result
}

func (h *mockHandler) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.events)
}

// TestDispatcher_RegisterAndDispatch tests basic registration and dispatching.
func TestDispatcher_RegisterAndDispatch(t *testing.T) {
	d := NewDispatcher(2, 10)
	handler := &mockHandler{}
	d.Register(handler)

	ctx := context.Background()
	d.Start(ctx)

	// Dispatch 5 events
	events := []*webhook.WebhookEvent{
		{ID: "1", Source: "github", EventType: "push", Payload: json.RawMessage(`{}`), CreatedAt: time.Now()},
		{ID: "2", Source: "github", EventType: "pull_request", Payload: json.RawMessage(`{}`), CreatedAt: time.Now()},
		{ID: "3", Source: "gitlab", EventType: "push", Payload: json.RawMessage(`{}`), CreatedAt: time.Now()},
		{ID: "4", Source: "gitlab", EventType: "merge_request", Payload: json.RawMessage(`{}`), CreatedAt: time.Now()},
		{ID: "5", Source: "github", EventType: "ping", Payload: json.RawMessage(`{}`), CreatedAt: time.Now()},
	}

	for _, event := range events {
		if err := d.Dispatch(event); err != nil {
			t.Fatalf("Failed to dispatch event: %v", err)
		}
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	d.Stop()

	// Verify all events were received
	receivedEvents := handler.getEvents()
	if len(receivedEvents) != 5 {
		t.Errorf("Expected 5 events, got %d", len(receivedEvents))
	}

	// Verify event IDs
	receivedIDs := make(map[string]bool)
	for _, e := range receivedEvents {
		receivedIDs[e.ID] = true
	}
	for _, e := range events {
		if !receivedIDs[e.ID] {
			t.Errorf("Event %s was not received", e.ID)
		}
	}
}

// TestDispatcher_Routing tests that handlers only receive events for their registered types.
func TestDispatcher_Routing(t *testing.T) {
	d := NewDispatcher(2, 10)
	
	// Handler that only handles "push" events
	pushHandler := &mockHandler{eventTypes: []string{"push"}}
	d.Register(pushHandler)

	ctx := context.Background()
	d.Start(ctx)

	// Dispatch push and ping events
	pushEvent := &webhook.WebhookEvent{
		ID:        "1",
		Source:    "github",
		EventType: "push",
		Payload:   json.RawMessage(`{}`),
		CreatedAt: time.Now(),
	}
	pingEvent := &webhook.WebhookEvent{
		ID:        "2",
		Source:    "github",
		EventType: "ping",
		Payload:   json.RawMessage(`{}`),
		CreatedAt: time.Now(),
	}

	if err := d.Dispatch(pushEvent); err != nil {
		t.Fatalf("Failed to dispatch push event: %v", err)
	}
	if err := d.Dispatch(pingEvent); err != nil {
		t.Fatalf("Failed to dispatch ping event: %v", err)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	d.Stop()

	// Verify only push event was received
	receivedEvents := pushHandler.getEvents()
	if len(receivedEvents) != 1 {
		t.Errorf("Expected 1 event, got %d", len(receivedEvents))
	}
	if len(receivedEvents) > 0 && receivedEvents[0].EventType != "push" {
		t.Errorf("Expected push event, got %s", receivedEvents[0].EventType)
	}
}

// TestDispatcher_QueueFull tests that dispatch returns error when queue is full.
func TestDispatcher_QueueFull(t *testing.T) {
	// Create dispatcher with queue size 1 and don't start workers
	d := NewDispatcher(0, 1)

	event1 := &webhook.WebhookEvent{
		ID:        "1",
		Source:    "github",
		EventType: "push",
		Payload:   json.RawMessage(`{}`),
		CreatedAt: time.Now(),
	}
	event2 := &webhook.WebhookEvent{
		ID:        "2",
		Source:    "github",
		EventType: "push",
		Payload:   json.RawMessage(`{}`),
		CreatedAt: time.Now(),
	}

	// First dispatch should succeed
	if err := d.Dispatch(event1); err != nil {
		t.Errorf("First dispatch should succeed, got error: %v", err)
	}

	// Second dispatch should fail with ErrQueueFull
	err := d.Dispatch(event2)
	if err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got: %v", err)
	}
}

// TestDispatcher_Stop tests that stop doesn't panic or hang.
func TestDispatcher_Stop(t *testing.T) {
	d := NewDispatcher(2, 10)
	handler := &mockHandler{}
	d.Register(handler)

	ctx := context.Background()
	d.Start(ctx)

	// Dispatch a few events
	for i := 0; i < 3; i++ {
		event := &webhook.WebhookEvent{
			ID:        string(rune('1' + i)),
			Source:    "github",
			EventType: "push",
			Payload:   json.RawMessage(`{}`),
			CreatedAt: time.Now(),
		}
		d.Dispatch(event)
	}

	// Stop should complete within reasonable time
	done := make(chan struct{})
	go func() {
		d.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success - stop completed
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() hung - did not complete within timeout")
	}
}

// TestDispatcher_HandlerError tests that handler errors are logged but don't stop processing.
func TestDispatcher_HandlerError(t *testing.T) {
	d := NewDispatcher(2, 10)
	
	// Handler that returns an error
	errorHandler := &mockHandler{err: errors.New("handler error")}
	d.Register(errorHandler)

	ctx := context.Background()
	d.Start(ctx)

	// Dispatch events
	for i := 0; i < 3; i++ {
		event := &webhook.WebhookEvent{
			ID:        string(rune('1' + i)),
			Source:    "github",
			EventType: "push",
			Payload:   json.RawMessage(`{}`),
			CreatedAt: time.Now(),
		}
		if err := d.Dispatch(event); err != nil {
			t.Fatalf("Failed to dispatch event: %v", err)
		}
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	d.Stop()

	// Verify all events were still processed despite errors
	if errorHandler.count() != 3 {
		t.Errorf("Expected 3 events processed, got %d", errorHandler.count())
	}
}

// TestDispatcher_MultipleHandlers tests multiple handlers receiving events.
func TestDispatcher_MultipleHandlers(t *testing.T) {
	d := NewDispatcher(2, 10)
	
	// Handler that accepts all events
	allHandler := &mockHandler{}
	// Handler that only accepts push events
	pushHandler := &mockHandler{eventTypes: []string{"push"}}
	
	d.Register(allHandler)
	d.Register(pushHandler)

	ctx := context.Background()
	d.Start(ctx)

	pushEvent := &webhook.WebhookEvent{
		ID:        "1",
		Source:    "github",
		EventType: "push",
		Payload:   json.RawMessage(`{}`),
		CreatedAt: time.Now(),
	}
	pingEvent := &webhook.WebhookEvent{
		ID:        "2",
		Source:    "github",
		EventType: "ping",
		Payload:   json.RawMessage(`{}`),
		CreatedAt: time.Now(),
	}

	d.Dispatch(pushEvent)
	d.Dispatch(pingEvent)

	time.Sleep(100 * time.Millisecond)
	d.Stop()

	// All handler should receive both events
	if allHandler.count() != 2 {
		t.Errorf("All handler expected 2 events, got %d", allHandler.count())
	}

	// Push handler should only receive push event
	if pushHandler.count() != 1 {
		t.Errorf("Push handler expected 1 event, got %d", pushHandler.count())
	}
	events := pushHandler.getEvents()
	if len(events) > 0 && events[0].EventType != "push" {
		t.Errorf("Expected push event, got %s", events[0].EventType)
	}
}
