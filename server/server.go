// Package server provides an HTTP server for webhook processing.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/webhookd/webhookd/crypto"
	"github.com/webhookd/webhookd/dispatcher"
	"github.com/webhookd/webhookd/store"
	"github.com/webhookd/webhookd/webhook"
)

// Config holds server configuration.
type Config struct {
	Port         int
	GitHubSecret string
	GitLabSecret string
	WorkerCount  int
	QueueSize    int
}

// Server is the HTTP server for webhook processing.
type Server struct {
	config     Config
	mux        *http.ServeMux
	store      store.EventStore
	dispatcher *dispatcher.Dispatcher
	parsers    map[string]webhook.Parser
	secrets    map[string]string // source -> secret
	httpServer *http.Server
}

// New creates a new Server with the given configuration.
func New(cfg Config) *Server {
	// Create memory store
	memStore := store.NewMemoryStore()

	// Create dispatcher
	disp := dispatcher.NewDispatcher(cfg.WorkerCount, cfg.QueueSize)

	// Register store handler
	storeHandler := &StoreHandler{store: memStore}
	disp.Register(storeHandler)

	// Create parsers map
	parsers := map[string]webhook.Parser{
		"github": webhook.NewGitHubParser(),
		"gitlab": webhook.NewGitLabParser(),
	}

	// Create secrets map
	secrets := map[string]string{
		"github": cfg.GitHubSecret,
		"gitlab": cfg.GitLabSecret,
	}

	s := &Server{
		config:     cfg,
		mux:        http.NewServeMux(),
		store:      memStore,
		dispatcher: disp,
		parsers:    parsers,
		secrets:    secrets,
	}

	// Setup routes
	s.mux.HandleFunc("/webhook/", s.handleWebhook)
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/events", s.handleEvents)

	return s
}

// Start starts the HTTP server and blocks until the context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	// Start dispatcher
	s.dispatcher.Start(ctx)

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: s.mux,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting HTTP server on %s", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Shutdown HTTP server with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("Shutting down HTTP server...")
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Stop dispatcher
	s.dispatcher.Stop()

	return nil
}

// ServeHTTP implements http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// handleWebhook processes incoming webhook requests.
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Extract source from URL path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		jsonError(w, "invalid webhook path", http.StatusNotFound)
		return
	}
	source := parts[1]

	// Check HTTP method
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Look up parser
	parser, ok := s.parsers[source]
	if !ok {
		jsonError(w, fmt.Sprintf("unknown webhook source: %s", source), http.StatusNotFound)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	// Verify signature if secret is configured
	secret, hasSecret := s.secrets[source]
	if hasSecret && secret != "" {
		if err := s.verifySignature(source, secret, r, body); err != nil {
			jsonError(w, "signature verification failed", http.StatusUnauthorized)
			return
		}
	}

	// Parse webhook
	event, err := parser.Parse(r, body)
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to parse webhook: %v", err), http.StatusBadRequest)
		return
	}

	// Dispatch event
	if err := s.dispatcher.Dispatch(event); err != nil {
		if err == dispatcher.ErrQueueFull {
			jsonError(w, "queue is full, please retry later", http.StatusServiceUnavailable)
			return
		}
		jsonError(w, fmt.Sprintf("failed to dispatch event: %v", err), http.StatusInternalServerError)
		return
	}

	// Return accepted response
	jsonOK(w, map[string]string{
		"id":     event.ID,
		"status": "accepted",
	}, http.StatusAccepted)
}

// verifySignature verifies the webhook signature based on the source.
func (s *Server) verifySignature(source, secret string, r *http.Request, body []byte) error {
	switch source {
	case "github":
		signature := r.Header.Get("X-Hub-Signature-256")
		if signature == "" {
			return fmt.Errorf("missing signature header")
		}
		// Strip "sha256=" prefix
		signature = strings.TrimPrefix(signature, "sha256=")
		return crypto.VerifySignature([]byte(secret), body, signature)

	case "gitlab":
		token := r.Header.Get("X-Gitlab-Token")
		if token != secret {
			return fmt.Errorf("invalid token")
		}
		return nil

	default:
		return nil
	}
}

// handleHealth returns the health status of the server.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	}, http.StatusOK)
}

// handleEvents returns a list of events based on query parameters.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	source := query.Get("source")
	eventType := query.Get("event_type")

	limit := 50
	if limitStr := query.Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
			if limit > 1000 {
				limit = 1000
			}
			if limit < 1 {
				limit = 1
			}
		}
	}

	// Build filter
	filter := store.EventFilter{
		Source:    source,
		EventType: eventType,
		Limit:     limit,
	}

	// Get events from store
	events, err := s.store.List(filter)
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to list events: %v", err), http.StatusInternalServerError)
		return
	}

	// Return events
	jsonOK(w, events, http.StatusOK)
}

// StoreHandler implements dispatcher.Handler and saves events to the store.
type StoreHandler struct {
	store store.EventStore
}

// EventTypes returns the event types this handler is interested in.
// Returning nil means this handler accepts all event types.
func (h *StoreHandler) EventTypes() []string {
	return nil
}

// Handle processes the webhook event by saving it to the store.
func (h *StoreHandler) Handle(ctx context.Context, event *webhook.WebhookEvent) error {
	return h.store.Save(event)
}

// jsonError writes a JSON error response.
func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}

// jsonOK writes a JSON success response.
func jsonOK(w http.ResponseWriter, data interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}
