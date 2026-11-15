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
	"github.com/ostendorf-cloud/ostendorf.dev/internal/webapp"
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

	handler, err := buildHandler(cfg)
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

// buildHandler serves the embedded SPA bundle's assets and the
// prerendered HTML per route, through internal/webapp. A real application
// adding a session, a form or a rate limiter wires govue's own middleware
// pieces in here too; see AGENTS.md.
func buildHandler(cfg config.Config) (http.Handler, error) {
	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, fmt.Errorf("website: %w", err)
	}
	prerendered, err := fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		return nil, fmt.Errorf("website: %w", err)
	}

	return webapp.NewHandler(webapp.Config{SiteURL: cfg.SiteURL}, dist, prerendered)
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
