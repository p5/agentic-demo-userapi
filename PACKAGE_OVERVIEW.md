# Package Overview

This document provides an overview of the two packages created: `store` and `dispatcher`.

## Package: store

**Location**: `/sandbox/workspace/store/store.go`

### Exported Types:

1. **EventFilter** - Defines criteria for filtering stored events
   - `Source string` - filter by source; empty means all
   - `EventType string` - filter by event type; empty means all
   - `Since time.Time` - only events after this time; zero means all
   - `Limit int` - max results; 0 means no limit

2. **EventStore** (interface) - Interface for persisting and retrieving webhook events
   - `Save(event *webhook.WebhookEvent) error` - Persists a webhook event
   - `Get(id string) (*webhook.WebhookEvent, error)` - Retrieves an event by ID
   - `List(filter EventFilter) ([]*webhook.WebhookEvent, error)` - Returns filtered events

3. **MemoryStore** - In-memory implementation of EventStore
   - Thread-safe using sync.RWMutex
   - Stores events in a map[string]*webhook.WebhookEvent

### Exported Functions:

- `NewMemoryStore() *MemoryStore` - Creates a new in-memory event store

### Exported Variables:

- `ErrEventNotFound` - Returned when an event cannot be found by ID
- `ErrDuplicateEvent` - Returned when attempting to save an event with a duplicate ID

### Key Features:

- Thread-safe operations using RWMutex
- Deep copy of events on Save to prevent external modifications
- Case-insensitive source filtering using strings.EqualFold
- Descending sort by CreatedAt timestamp
- Support for limit on returned results

## Package: dispatcher

**Location**: `/sandbox/workspace/dispatcher/dispatcher.go`

### Exported Types:

1. **Handler** (interface) - Processes webhook events
   - `EventTypes() []string` - Returns event types this handler is interested in
   - `Handle(ctx context.Context, event *webhook.WebhookEvent) error` - Processes the event

2. **Dispatcher** - Routes webhook events to registered handlers using a worker pool
   - Uses worker pool pattern for concurrent processing
   - Non-blocking dispatch with bounded queue

### Exported Functions:

- `NewDispatcher(workerCount int, queueSize int) *Dispatcher` - Creates a new dispatcher
  - workerCount defaults to 4 if <= 0
  - queueSize defaults to 100 if <= 0
- `(d *Dispatcher) Register(h Handler)` - Adds a handler
- `(d *Dispatcher) Start(ctx context.Context)` - Launches worker pool
- `(d *Dispatcher) Dispatch(event *webhook.WebhookEvent) error` - Sends event to queue
- `(d *Dispatcher) Stop()` - Gracefully shuts down dispatcher

### Exported Variables:

- `ErrQueueFull` - Returned when the dispatcher's queue is full

### Key Features:

- Worker pool pattern for concurrent event processing
- Selective event routing based on Handler.EventTypes()
- Non-blocking dispatch with error on queue full
- Graceful shutdown with WaitGroup
- Context-aware workers that exit on context cancellation
- Error logging for handler failures (doesn't stop processing)

## Tests

**Location**: `/sandbox/workspace/dispatcher/dispatcher_test.go`

### Test Cases:

1. **TestDispatcher_RegisterAndDispatch** - Basic registration and dispatching
2. **TestDispatcher_Routing** - Handler filtering by event type
3. **TestDispatcher_QueueFull** - Queue overflow handling
4. **TestDispatcher_Stop** - Graceful shutdown
5. **TestDispatcher_HandlerError** - Error handling doesn't stop processing
6. **TestDispatcher_MultipleHandlers** - Multiple handlers receiving events

All tests use a mockHandler that safely records received events using sync.Mutex.

## Dependencies

Both packages depend only on:
- Standard library packages (sync, time, strings, sort, context, log, errors)
- `github.com/webhookd/webhookd/webhook` package (upstream dependency)

No external third-party dependencies required.
