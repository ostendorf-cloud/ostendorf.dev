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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return runApp(ctx, cfg, logger, httpserver.RunAdmin, nil)
}

// serveFunc is the shape of httpserver.RunAdmin, so a test can stand in for
// the two listeners and still go through runApp's startup and shutdown.
type serveFunc func(context.Context, httpserver.Config, httpserver.AdminConfig) error

// runApp builds the page view counter and the handlers, serves them and,
// once serve returns, sends the page views still queued. umamiHTTP is the
// client that posts to Umami: nil means govue's default, a test passes a
// fake transport.
func runApp(ctx context.Context, cfg config.Config, logger *slog.Logger, serve serveFunc, umamiHTTP *http.Client) error {
	// One Prometheus registry serves the scrape and holds the dropped view
	// counter, so a lost view shows up in /metrics.
	promRegistry := metrics.NewRegistry()
	views, err := buildViews(cfg, logger, umamiHTTP, metrics.NewCounter(promRegistry, "analytics_views_dropped_total",
		"Page views lost to a full queue or a shutdown before they reached Umami."))
	if err != nil {
		return err
	}
	logViewsStatus(logger, cfg, views)

	handler, err := buildHandler(cfg, logger, views)
	if err != nil {
		_ = views.Close(context.Background())
		return err
	}

	registry := health.New()

	runErr := serve(ctx, httpserver.Config{
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
// comes from the trusted proxy list a deployment behind a proxy needs. The
// whole address is never sent: FullIP stays off. httpClient posts to Umami,
// nil is govue's default. onDrop is called for every view that is lost and
// must not block.
func buildViews(cfg config.Config, logger *slog.Logger, httpClient *http.Client, onDrop func()) (*analytics.Client, error) {
	return analytics.New(cfg.Umami, analytics.Options{
		Hostname:   cfg.SiteURL.Hostname(),
		Logger:     logger,
		HTTPClient: httpClient,
		ClientIP:   newResolver(cfg),
		OnDrop:     onDrop,
	})
}

// logViewsStatus says once at startup whether page views are counted: govue
// logs nothing when counting is off. Counting on with no trusted proxy is a
// warning, because behind an ingress every request then comes from the
// proxy's private address and nothing is counted, silently.
func logViewsStatus(logger *slog.Logger, cfg config.Config, views *analytics.Client) {
	logger.Info("page view counting", "enabled", views.Enabled())
	if views.Enabled() && len(cfg.TrustedProxies) == 0 {
		logger.Warn("page view counting is on but TRUSTED_PROXIES is empty: behind a proxy no view is counted")
	}
}

// newResolver is the one client address resolver behind TRUSTED_PROXIES. The
// page view counter and the access log both use it.
func newResolver(cfg config.Config) *clientip.Resolver {
	return clientip.NewResolver(cfg.TrustedProxies)
}

// buildHandler serves the embedded SPA bundle's assets and the
// prerendered HTML per route, through internal/webapp, with views counting
// the pages and logger receiving the access log. A real application adding
// a session, a form or a rate limiter wires govue's own middleware pieces
// in here too, but never between the mux and views, which reads the route
// pattern the mux sets; see AGENTS.md.
func buildHandler(cfg config.Config, logger *slog.Logger, views *analytics.Client) (http.Handler, error) {
	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, fmt.Errorf("website: %w", err)
	}
	prerendered, err := fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		return nil, fmt.Errorf("website: %w", err)
	}

	return webapp.NewHandler(webapp.Config{
		SiteURL:  cfg.SiteURL,
		Views:    views,
		Logger:   logger,
		ClientIP: newResolver(cfg),
	}, dist, prerendered)
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
