package content

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ostendorf-cloud/cv/client"

	"github.com/ostendorf-cloud/ostendorf.dev/internal/config"
)

// FromConfig builds the Store over one cv client per language. httpClient is
// the client the calls go through: nil means the cv client's default, a test
// passes a fake transport.
//
// With no CVD_URL it logs one info line and returns a Store with no source,
// so the site starts and every data page answers the placeholder. A bad
// address cannot reach this point: config.Load rejects it.
func FromConfig(cfg config.CVD, httpClient *http.Client, logger *slog.Logger) (*Store, error) {
	opts := Options{
		TTL:     cfg.CacheTTL,
		Timeout: cfg.Timeout,
		Logger:  logger,
	}
	if cfg.URL == nil {
		logger.Info("cvd is not configured: the data pages answer the placeholder")
		return New(opts), nil
	}

	opts.Sources = make(map[string]Source, 2)
	for _, lang := range []string{LangDE, LangEN} {
		clientOpts := []client.Option{
			client.WithBaseURL(cfg.URL.String()),
			client.WithLanguage(lang),
			client.WithTimeout(cfg.Timeout),
		}
		if httpClient != nil {
			clientOpts = append(clientOpts, client.WithHTTPClient(httpClient))
		}
		c, err := client.New(clientOpts...)
		if err != nil {
			return nil, fmt.Errorf("content: build the %s client: %w", lang, err)
		}
		opts.Sources[lang] = c
	}
	return New(opts), nil
}
