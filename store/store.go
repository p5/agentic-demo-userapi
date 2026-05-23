package store

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/webhookd/webhookd/webhook"
)

// EventFilter defines criteria for filtering stored events.
type EventFilter struct {
	Source    string    // filter by source; empty means all
	EventType string    // filter by event type; empty means all
	Since     time.Time // only events after this time; zero means all
	Limit     int       // max results; 0 means no limit
}

// EventStore is the interface for persisting and retrieving webhook events.
type EventStore interface {
	// Save persists a webhook event. Returns an error if the event ID already exists.
	Save(event *webhook.WebhookEvent) error
	// Get retrieves an event by ID. Returns (nil, ErrEventNotFound) if not found.
	Get(id string) (*webhook.WebhookEvent, error)
	// List returns events matching the filter, ordered by CreatedAt descending.
	List(filter EventFilter) ([]*webhook.WebhookEvent, error)
}

var (
	// ErrEventNotFound is returned when an event cannot be found by ID.
	ErrEventNotFound = errors.New("store: event not found")
	// ErrDuplicateEvent is returned when attempting to save an event with a duplicate ID.
	ErrDuplicateEvent = errors.New("store: duplicate event ID")
)

// MemoryStore implements EventStore with an in-memory map protected by sync.RWMutex.
type MemoryStore struct {
	mu     sync.RWMutex
	events map[string]*webhook.WebhookEvent
}

// NewMemoryStore creates a new in-memory event store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		events: make(map[string]*webhook.WebhookEvent),
	}
}

// Save persists a webhook event. Returns an error if the event ID already exists.
func (s *MemoryStore) Save(event *webhook.WebhookEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.events[event.ID]; exists {
		return ErrDuplicateEvent
	}

	// Store a copy of the event
	eventCopy := &webhook.WebhookEvent{
		ID:        event.ID,
		Source:    event.Source,
		EventType: event.EventType,
		Payload:   event.Payload,
		Headers:   make(map[string]string),
		CreatedAt: event.CreatedAt,
	}
	for k, v := range event.Headers {
		eventCopy.Headers[k] = v
	}

	s.events[event.ID] = eventCopy
	return nil
}

// Get retrieves an event by ID. Returns (nil, ErrEventNotFound) if not found.
func (s *MemoryStore) Get(id string) (*webhook.WebhookEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	event, exists := s.events[id]
	if !exists {
		return nil, ErrEventNotFound
	}

	return event, nil
}

// List returns events matching the filter, ordered by CreatedAt descending.
func (s *MemoryStore) List(filter EventFilter) ([]*webhook.WebhookEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*webhook.WebhookEvent

	// Apply filters
	for _, event := range s.events {
		// Filter by Source (case-insensitive)
		if filter.Source != "" && !strings.EqualFold(event.Source, filter.Source) {
			continue
		}

		// Filter by EventType
		if filter.EventType != "" && event.EventType != filter.EventType {
			continue
		}

		// Filter by Since
		if !filter.Since.IsZero() && event.CreatedAt.Before(filter.Since) {
			continue
		}

		results = append(results, event)
	}

	// Sort by CreatedAt descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	// Apply Limit
	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}

	return results, nil
}
