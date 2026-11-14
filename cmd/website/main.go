package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ostendorf-cloud/govue/health"
	"github.com/ostendorf-cloud/govue/httpserver"
	"github.com/ostendorf-cloud/govue/logging"
	"github.com/ostendorf-cloud/govue/metrics"

	"github.com/ostendorf-cloud/ostendorf.dev/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	logger := logging.New(os.Stdout, cfg.LogLevel)

	handler, err := buildHandler()
	if err != nil {
		return err
	}

	registry := health.New()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return httpserver.RunAdmin(ctx, httpserver.Config{
		Addr:    cfg.Addr,
		Handler: handler,
		Logger:  logger,
		// Failing the readiness probe before the drain delay starts is what
		// lets a load balancer stop sending new requests before the server
		// stops accepting connections at all.
		BeforeShutdown: registry.Drain,
	}, httpserver.AdminConfig{
		Addr:    cfg.AdminAddr,
		Handler: buildAdminHandler(registry),
	})
}

// buildHandler serves the embedded SPA bundle. A real application replaces
// this with its own handler-building package, adding its own API routes and
// whatever middleware govue pieces it needs; see AGENTS.md.
func buildHandler() (http.Handler, error) {
	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, fmt.Errorf("website: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServerFS(dist))
	return mux, nil
}

// buildAdminHandler serves govue's admin listener: liveness, readiness and
// the Prometheus scrape, reached only from inside the cluster, never through
// this chart's own Ingress.
func buildAdminHandler(registry *health.Registry) http.Handler {
	mux := http.NewServeMux()
	registry.Handle(mux)
	mux.Handle("GET /metrics", metrics.Handler(metrics.NewRegistry()))
	return mux
}
