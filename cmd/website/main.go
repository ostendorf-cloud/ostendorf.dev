package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ostendorf-cloud/govue/analytics"
	"github.com/ostendorf-cloud/govue/clientip"
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

	// One Prometheus registry serves the scrape and holds the dropped view
	// counter, so a lost view shows up in /metrics.
	promRegistry := metrics.NewRegistry()
	views, err := buildViews(cfg, logger, metrics.NewCounter(promRegistry, "analytics_views_dropped_total",
		"Page views lost to a full queue or a shutdown before they reached Umami."))
	if err != nil {
		return err
	}
	// govue logs nothing when counting is off, so this line is the one
	// startup statement of whether page views are counted.
	logger.Info("page view counting", "enabled", views.Enabled())

	handler, err := buildHandler(cfg, views)
	if err != nil {
		_ = views.Close(context.Background())
		return err
	}

	registry := health.New()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runErr := httpserver.RunAdmin(ctx, httpserver.Config{
		Addr:    cfg.Addr,
		Handler: handler,
		Logger:  logger,
		// Failing the readiness probe before the drain delay starts is what
		// lets a load balancer stop sending new requests before the server
		// stops accepting connections at all.
		BeforeShutdown: registry.Drain,
	}, httpserver.AdminConfig{
		Addr:    cfg.AdminAddr,
		Handler: buildAdminHandler(registry, metrics.Handler(promRegistry)),
	})

	// The server has stopped, so what is still queued is sent now, within a
	// bound, on a fresh context: ctx has ended already.
	closeCtx, cancel := context.WithTimeout(context.Background(), viewsCloseTimeout)
	defer cancel()
	return errors.Join(runErr, views.Close(closeCtx))
}

// viewsCloseTimeout bounds how long shutdown waits for queued page views.
const viewsCloseTimeout = 5 * time.Second

// buildViews builds the server side page view counter. It counts nothing
// unless both UMAMI_URL and UMAMI_WEBSITE_ID are set. The client address
// comes from the trusted proxy list a deployment behind a proxy needs.
// onDrop is called for every view that is lost and must not block.
func buildViews(cfg config.Config, logger *slog.Logger, onDrop func()) (*analytics.Client, error) {
	return analytics.New(cfg.Umami, analytics.Options{
		Hostname: cfg.SiteURL.Hostname(),
		Logger:   logger,
		ClientIP: clientip.NewResolver(cfg.TrustedProxies),
		OnDrop:   onDrop,
	})
}

// buildHandler serves the embedded SPA bundle's assets and the
// prerendered HTML per route, through internal/webapp, with views counting
// the pages. A real application adding a session, a form or a rate limiter
// wires govue's own middleware pieces in here too, but never between the
// mux and views, which reads the route pattern the mux sets; see AGENTS.md.
func buildHandler(cfg config.Config, views *analytics.Client) (http.Handler, error) {
	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, fmt.Errorf("website: %w", err)
	}
	prerendered, err := fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		return nil, fmt.Errorf("website: %w", err)
	}

	return webapp.NewHandler(webapp.Config{SiteURL: cfg.SiteURL, Views: views}, dist, prerendered)
}

// buildAdminHandler serves govue's admin listener: liveness, readiness and
// the Prometheus scrape, reached only from inside the cluster, never through
// this chart's own Ingress.
func buildAdminHandler(registry *health.Registry, metricsHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	registry.Handle(mux)
	mux.Handle("GET /metrics", metricsHandler)
	return mux
}
