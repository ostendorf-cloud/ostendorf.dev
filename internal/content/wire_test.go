package content

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ostendorf-cloud/ostendorf.dev/internal/config"
)

// fakeCVD stands in for cvd. mode picks what every request gets.
type fakeCVD struct {
	*httptest.Server
	hits atomic.Int32
}

const (
	modeOK = iota
	modeServerError
	modeNewSchema
	modeMalformed
	modeWrongLanguage
	modeSlow
)

func newFakeCVD(t *testing.T, mode int) *fakeCVD {
	t.Helper()

	f := &fakeCVD{}
	release := make(chan struct{})
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		lang := r.URL.Query().Get("lang")
		w.Header().Set("Content-Type", "application/json")
		switch mode {
		case modeOK:
			w.Header().Set("ETag", `"abc123"`)
			fmt.Fprintf(w, `{"schema_version":1,"language":%q,"projects":[{"id":"cv-%s"}]}`, lang, lang)
		case modeServerError:
			w.WriteHeader(http.StatusBadGateway)
		case modeNewSchema:
			fmt.Fprintf(w, `{"schema_version":2,"language":%q}`, lang)
		case modeMalformed:
			fmt.Fprint(w, `{"schema_version":`)
		case modeWrongLanguage:
			fmt.Fprint(w, `{"schema_version":1,"language":"fr"}`)
		case modeSlow:
			select {
			case <-release:
			case <-r.Context().Done():
			}
		}
	}))
	t.Cleanup(func() {
		close(release)
		f.Close()
	})
	return f
}

func cvdConfig(t *testing.T, raw string, timeout time.Duration) config.CVD {
	t.Helper()

	cfg := config.CVD{CacheTTL: time.Minute, Timeout: timeout}
	if raw != "" {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("url.Parse(%q) error = %v", raw, err)
		}
		cfg.URL = u
	}
	return cfg
}

func TestFromConfig_withoutAnAddress(t *testing.T) {
	t.Parallel()

	log := &recorder{}
	s, err := FromConfig(cvdConfig(t, "", time.Second), nil, slog.New(log))
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	for _, lang := range []string{LangDE, LangEN} {
		if _, err := s.Get(t.Context(), lang); !errors.Is(err, ErrUnavailable) {
			t.Errorf("Get(%s) error = %v, want ErrUnavailable", lang, err)
		}
	}
	records := log.all()
	if len(records) != 1 || records[0].level != slog.LevelInfo {
		t.Errorf("log lines = %+v, want exactly one info line", records)
	}
}

func TestFromConfig_readsEachLanguageFromCVD(t *testing.T) {
	t.Parallel()

	srv := newFakeCVD(t, modeOK)
	s, err := FromConfig(cvdConfig(t, srv.URL, 2*time.Second), srv.Client(), slog.New(&recorder{}))
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	for _, lang := range []string{LangDE, LangEN} {
		doc, err := s.Get(t.Context(), lang)
		if err != nil {
			t.Fatalf("Get(%s) error = %v", lang, err)
		}
		if doc.Language != lang || doc.Projects[0].ID != "cv-"+lang {
			t.Errorf("Get(%s) = language %q project %q, want the %s document", lang, doc.Language, doc.Projects[0].ID, lang)
		}
	}
}

func TestFromConfig_aBadAnswerIsNoDocument(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		mode int
		// timeout is the bound of one read.
		timeout time.Duration
	}{
		{name: "a 5xx status", mode: modeServerError, timeout: 2 * time.Second},
		{name: "an unknown schema version", mode: modeNewSchema, timeout: 2 * time.Second},
		{name: "a body that is no JSON", mode: modeMalformed, timeout: 2 * time.Second},
		{name: "a document of another language", mode: modeWrongLanguage, timeout: 2 * time.Second},
		{name: "no answer before the timeout", mode: modeSlow, timeout: 50 * time.Millisecond},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := newFakeCVD(t, tc.mode)
			log := &recorder{}
			s, err := FromConfig(cvdConfig(t, srv.URL, tc.timeout), srv.Client(), slog.New(log))
			if err != nil {
				t.Fatalf("FromConfig() error = %v", err)
			}

			if _, err := s.Get(t.Context(), LangEN); !errors.Is(err, ErrUnavailable) {
				t.Fatalf("Get() error = %v, want ErrUnavailable", err)
			}
			s.refreshes.Wait()

			records := log.all()
			if len(records) != 1 {
				t.Fatalf("log lines = %d, want 1 for the failed read", len(records))
			}
			for _, attr := range records[0].attrs {
				if text, ok := attr.(string); ok && text == srv.URL {
					t.Errorf("log line carries the address of cvd: %+v", records[0])
				}
			}
			if got := srv.hits.Load(); got != 1 {
				t.Errorf("requests to cvd = %d, want 1", got)
			}
		})
	}
}

func TestFromConfig_aSchemaRefusalIsLoggedWithBothVersions(t *testing.T) {
	t.Parallel()

	srv := newFakeCVD(t, modeNewSchema)
	log := &recorder{}
	s, err := FromConfig(cvdConfig(t, srv.URL, 2*time.Second), srv.Client(), slog.New(log))
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	if _, err := s.Get(t.Context(), LangEN); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Get() error = %v, want ErrUnavailable", err)
	}
	s.refreshes.Wait()

	records := log.all()
	if len(records) != 1 || records[0].level != slog.LevelError {
		t.Fatalf("log lines = %+v, want one error line", records)
	}
	if records[0].attrs["got"] != int64(2) || records[0].attrs["supported"] != int64(1) {
		t.Errorf("log attrs = %+v, want got 2 and supported 1", records[0].attrs)
	}
}
