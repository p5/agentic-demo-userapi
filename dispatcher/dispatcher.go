package dispatcher

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/webhookd/webhookd/webhook"
)

// Handler processes a webhook event. Implementations must be safe for concurrent use.
type Handler interface {
	// EventTypes returns the event types this handler is interested in. Empty slice means all.
	EventTypes() []string
	// Handle processes the event. Returns error on failure.
	Handle(ctx context.Context, event *webhook.WebhookEvent) error
}

// Dispatcher routes webhook events to registered handlers using a worker pool.
type Dispatcher struct {
	handlers    []Handler
	queue       chan *webhook.WebhookEvent
	wg          sync.WaitGroup
	quit        chan struct{}
	workerCount int
}

var (
	// ErrQueueFull is returned when the dispatcher's queue is full.
	ErrQueueFull = errors.New("dispatcher: queue is full")
)

// NewDispatcher creates a new Dispatcher with the specified worker pool size and queue capacity.
// workerCount defaults to 4 if <= 0, queueSize defaults to 100 if <= 0.
func NewDispatcher(workerCount int, queueSize int) *Dispatcher {
	if workerCount <= 0 {
		workerCount = 4
	}
	if queueSize <= 0 {
		queueSize = 100
	}

	return &Dispatcher{
		handlers:    make([]Handler, 0),
		queue:       make(chan *webhook.WebhookEvent, queueSize),
		quit:        make(chan struct{}),
		workerCount: workerCount,
	}
}

// Register adds a handler to the dispatcher's handler list.
func (d *Dispatcher) Register(h Handler) {
	d.handlers = append(d.handlers, h)
}

// Start launches the worker pool to process events from the queue.
// Workers will continue processing until Stop is called or the context is cancelled.
func (d *Dispatcher) Start(ctx context.Context) {
	for i := 0; i < d.workerCount; i++ {
		d.wg.Add(1)
		go d.worker(ctx)
	}
}

func (d *Dispatcher) worker(ctx context.Context) {
	defer d.wg.Done()

	for {
		select {
		case <-d.quit:
			return
		case <-ctx.Done():
			return
		case event, ok := <-d.queue:
			if !ok {
				return
			}
			d.processEvent(ctx, event)
		}
	}
}

func (d *Dispatcher) processEvent(ctx context.Context, event *webhook.WebhookEvent) {
	for _, handler := range d.handlers {
		eventTypes := handler.EventTypes()
		
		// If handler accepts all events or this specific event type
		if len(eventTypes) == 0 || d.containsEventType(eventTypes, event.EventType) {
			if err := handler.Handle(ctx, event); err != nil {
				log.Printf("dispatcher: handler error for event %s (type: %s): %v",
					event.ID, event.EventType, err)
			}
		}
	}
}

func (d *Dispatcher) containsEventType(types []string, eventType string) bool {
	for _, t := range types {
		if t == eventType {
			return true
		}
	}
	return false
}

// Dispatch sends an event to the queue for processing.
// Returns ErrQueueFull if the queue is at capacity.
func (d *Dispatcher) Dispatch(event *webhook.WebhookEvent) error {
	select {
	case d.queue <- event:
		return nil
	default:
		return ErrQueueFull
	}
}

// Stop gracefully shuts down the dispatcher, waiting for all workers to finish processing.
func (d *Dispatcher) Stop() {
	close(d.quit)
	d.wg.Wait()
}
