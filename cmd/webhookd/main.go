// Package main provides the CLI entry point for webhookd.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/webhookd/webhookd/server"
)

func main() {
	// Define flags
	var (
		port         = flag.Int("port", 8080, "HTTP server port")
		githubSecret = flag.String("github-secret", "", "GitHub webhook secret for signature verification")
		gitlabSecret = flag.String("gitlab-secret", "", "GitLab webhook secret for token verification")
		workers      = flag.Int("workers", 4, "Number of worker goroutines for event processing")
		queueSize    = flag.Int("queue-size", 100, "Size of the event queue")
	)

	// Check for subcommand
	if len(os.Args) > 1 && os.Args[1] == "start" {
		// Parse flags from position 2 onwards
		flag.CommandLine.Parse(os.Args[2:])
	} else {
		// Parse all flags
		flag.Parse()
	}

	// Create server configuration
	cfg := server.Config{
		Port:         *port,
		GitHubSecret: *githubSecret,
		GitLabSecret: *gitlabSecret,
		WorkerCount:  *workers,
		QueueSize:    *queueSize,
	}

	// Create context that cancels on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create server
	srv := server.New(cfg)

	// Print startup banner
	fmt.Printf("webhookd starting on :%d\n", cfg.Port)
	fmt.Printf("  Workers: %d\n", cfg.WorkerCount)
	fmt.Printf("  Queue Size: %d\n", cfg.QueueSize)
	if cfg.GitHubSecret != "" {
		fmt.Println("  GitHub signature verification: enabled")
	}
	if cfg.GitLabSecret != "" {
		fmt.Println("  GitLab token verification: enabled")
	}

	// Start server (blocks until context is cancelled)
	if err := srv.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("webhookd stopped")
}
