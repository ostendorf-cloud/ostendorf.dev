package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DefaultGitHubAPI is the REST API's base URL.
const DefaultGitHubAPI = "https://api.github.com"

// maxBody caps what one API response may hold. A repository's metadata is a few
// kilobytes, so anything near this is not what was asked for.
const maxBody = 4 << 20

// Repo is the GitHub data one project page shows. It holds public data only.
type Repo struct {
	FullName      string          `json:"full_name"`
	URL           string          `json:"url"`
	Description   *string         `json:"description"`
	Stars         int             `json:"stars"`
	Forks         int             `json:"forks"`
	Language      *string         `json:"language"`
	Languages     []LanguageShare `json:"languages"`
	Topics        []string        `json:"topics"`
	License       *string         `json:"license"`
	Archived      bool            `json:"archived"`
	PushedAt      string          `json:"pushed_at"`
	LatestRelease *Release        `json:"latest_release"`
}

// LanguageShare is the number of bytes of one language in a repository. The
// page works out percentages.
type LanguageShare struct {
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
}

// Release is a repository's latest published, non prerelease release.
type Release struct {
	Tag         string `json:"tag"`
	URL         string `json:"url"`
	PublishedAt string `json:"published_at"`
}

// ErrNotFound is returned for a repository GitHub does not show to the caller.
// A private repository answers the same way, which is the point: the site only
// lists public repositories.
var ErrNotFound = errors.New("not found")

// RateLimitError is returned when GitHub limits the caller and the wait it asks
// for is longer than the client is willing to sit through.
type RateLimitError struct {
	Path string
	Wait time.Duration
}

// Error implements error.
func (e *RateLimitError) Error() string {
	return fmt.Sprintf("github: GET %s: rate limited, GitHub asks for a wait of %s", e.Path, e.Wait.Round(time.Second))
}

// Client calls the GitHub REST API for public repository data. The token is
// optional: public data needs no scope, and a token only raises the rate limit.
// The token is sent in one header and is never put in an error, a log line or a
// URL.
type Client struct {
	// BaseURL is the API root, DefaultGitHubAPI when empty.
	BaseURL string
	// HTTP is the transport, a client with a 15 second timeout when nil.
	HTTP *http.Client
	// Token is the bearer token, none when empty.
	Token string
	// Attempts is how many times one request is tried, 4 when zero.
	Attempts int
	// MaxWait is the longest single wait for a rate limit or a retry, two minutes
	// when zero. A longer wait fails the build.
	MaxWait time.Duration
	// Sleep waits for d or until ctx ends. Tests replace it.
	Sleep func(ctx context.Context, d time.Duration) error
	// Now is the clock the rate limit reset time is compared to.
	Now func() time.Time
	// Log receives one line per retry, never a token.
	Log *slog.Logger
}

func (c *Client) base() string {
	if c.BaseURL == "" {
		return DefaultGitHubAPI
	}
	return strings.TrimRight(c.BaseURL, "/")
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP == nil {
		return &http.Client{Timeout: 15 * time.Second}
	}
	return c.HTTP
}

func (c *Client) attempts() int {
	if c.Attempts <= 0 {
		return 4
	}
	return c.Attempts
}

func (c *Client) maxWait() time.Duration {
	if c.MaxWait <= 0 {
		return 2 * time.Minute
	}
	return c.MaxWait
}

func (c *Client) sleep(ctx context.Context, d time.Duration) error {
	if c.Sleep != nil {
		return c.Sleep(ctx, d)
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *Client) logger() *slog.Logger {
	if c.Log == nil {
		return slog.New(slog.DiscardHandler)
	}
	return c.Log
}

// FetchRepos fetches every named repository, in sorted order. It stops at the
// first failure: a partial result is never returned.
func (c *Client) FetchRepos(ctx context.Context, names []string) (map[string]*Repo, error) {
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	out := make(map[string]*Repo, len(sorted))
	for _, name := range sorted {
		if _, done := out[name]; done {
			continue
		}
		repo, err := c.FetchRepo(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("github repository %s: %w", name, err)
		}
		out[name] = repo
	}
	return out, nil
}

// FetchRepo fetches the metadata of one public repository: the repository
// itself, its languages and its latest release.
func (c *Client) FetchRepo(ctx context.Context, name string) (*Repo, error) {
	if !repoPattern.MatchString(name) {
		return nil, fmt.Errorf("%q is not owner/name", name)
	}
	base := "/repos/" + name

	var meta struct {
		FullName    string   `json:"full_name"`
		HTMLURL     string   `json:"html_url"`
		Description *string  `json:"description"`
		Stars       int      `json:"stargazers_count"`
		Forks       int      `json:"forks_count"`
		Language    *string  `json:"language"`
		Topics      []string `json:"topics"`
		License     *struct {
			SPDXID *string `json:"spdx_id"`
		} `json:"license"`
		Archived bool   `json:"archived"`
		Private  bool   `json:"private"`
		PushedAt string `json:"pushed_at"`
	}
	if err := c.getJSON(ctx, base, &meta); err != nil {
		return nil, err
	}
	if meta.Private {
		return nil, fmt.Errorf("GET %s: repository is private, the site lists public repositories only", base)
	}
	if meta.FullName == "" || meta.PushedAt == "" {
		return nil, fmt.Errorf("GET %s: response lacks full_name or pushed_at", base)
	}

	var langs map[string]int64
	if err := c.getJSON(ctx, base+"/languages", &langs); err != nil {
		return nil, err
	}
	shares := make([]LanguageShare, 0, len(langs))
	for n, b := range langs {
		shares = append(shares, LanguageShare{Name: n, Bytes: b})
	}
	sort.Slice(shares, func(i, j int) bool {
		if shares[i].Bytes != shares[j].Bytes {
			return shares[i].Bytes > shares[j].Bytes
		}
		return shares[i].Name < shares[j].Name
	})

	var rel *Release
	var latest struct {
		Tag         string `json:"tag_name"`
		HTMLURL     string `json:"html_url"`
		PublishedAt string `json:"published_at"`
	}
	switch err := c.getJSON(ctx, base+"/releases/latest", &latest); {
	case errors.Is(err, ErrNotFound):
		// A repository without a release answers 404 here. That is data, not a failure.
	case err != nil:
		return nil, err
	default:
		rel = &Release{Tag: latest.Tag, URL: latest.HTMLURL, PublishedAt: latest.PublishedAt}
	}

	topics := append([]string{}, meta.Topics...)
	sort.Strings(topics)
	repo := &Repo{
		FullName:      meta.FullName,
		URL:           meta.HTMLURL,
		Description:   meta.Description,
		Stars:         meta.Stars,
		Forks:         meta.Forks,
		Language:      meta.Language,
		Languages:     shares,
		Topics:        topics,
		Archived:      meta.Archived,
		PushedAt:      meta.PushedAt,
		LatestRelease: rel,
	}
	if meta.License != nil && meta.License.SPDXID != nil && *meta.License.SPDXID != "NOASSERTION" {
		repo.License = meta.License.SPDXID
	}
	return repo, nil
}

func (c *Client) getJSON(ctx context.Context, path string, into any) error {
	body, err := c.get(ctx, path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, into); err != nil {
		return fmt.Errorf("GET %s: malformed response: %w", path, err)
	}
	return nil
}

// get performs one GET with retries. 5xx and transport errors back off and
// retry. A rate limit answer (429, or 403 with Retry-After or an exhausted
// quota) waits as long as GitHub asks when that is within MaxWait and fails
// otherwise. Every other status fails at once.
func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	var lastErr error
	for attempt := 1; attempt <= c.attempts(); attempt++ {
		body, wait, err := c.once(ctx, path)
		if err == nil {
			return body, nil
		}
		if ctx.Err() != nil {
			return nil, fmt.Errorf("GET %s: %w", path, ctx.Err())
		}
		if wait < 0 {
			return nil, err
		}
		lastErr = err
		if attempt == c.attempts() {
			break
		}
		if wait == 0 {
			wait = time.Duration(1<<(attempt-1)) * time.Second
		}
		if wait > c.maxWait() {
			return nil, &RateLimitError{Path: path, Wait: wait}
		}
		c.logger().Warn("github request failed, retrying", "path", path, "attempt", attempt, "wait", wait.String(), "reason", err.Error())
		if err := c.sleep(ctx, wait); err != nil {
			return nil, fmt.Errorf("GET %s: %w", path, err)
		}
	}
	return nil, fmt.Errorf("giving up after %d attempts: %w", c.attempts(), lastErr)
}

// once makes one request. wait is the delay before a retry: zero asks for the
// default backoff, negative means the error is final.
func (c *Client) once(ctx context.Context, path string) (body []byte, wait time.Duration, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+path, nil)
	if err != nil {
		return nil, -1, fmt.Errorf("GET %s: %w", path, err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "ostendorf.dev-content-build")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		// A timeout or a refused connection can pass, so it retries.
		return nil, 0, fmt.Errorf("GET %s: %s", path, scrub(err.Error(), c.Token))
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusOK:
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
		if err != nil {
			return nil, 0, fmt.Errorf("GET %s: reading body: %s", path, scrub(err.Error(), c.Token))
		}
		if len(data) > maxBody {
			return nil, -1, fmt.Errorf("GET %s: response larger than %d bytes", path, maxBody)
		}
		return data, 0, nil
	case resp.StatusCode == http.StatusNotFound:
		return nil, -1, fmt.Errorf("GET %s: %w", path, ErrNotFound)
	case resp.StatusCode == http.StatusTooManyRequests, resp.StatusCode == http.StatusForbidden:
		if d, ok := c.rateLimitWait(resp.Header); ok {
			return nil, d, fmt.Errorf("GET %s: status %d, rate limited", path, resp.StatusCode)
		}
		return nil, -1, fmt.Errorf("GET %s: status %d", path, resp.StatusCode)
	case resp.StatusCode >= 500:
		return nil, 0, fmt.Errorf("GET %s: status %d", path, resp.StatusCode)
	default:
		return nil, -1, fmt.Errorf("GET %s: status %d", path, resp.StatusCode)
	}
}

// rateLimitWait reads how long GitHub wants the caller to wait. It reports
// false for a 403 that is not a rate limit.
func (c *Client) rateLimitWait(h http.Header) (time.Duration, bool) {
	if v := h.Get("Retry-After"); v != "" {
		secs, err := strconv.Atoi(v)
		if err != nil || secs < 0 {
			return 0, false
		}
		if secs == 0 {
			secs = 1
		}
		return time.Duration(secs) * time.Second, true
	}
	if h.Get("X-RateLimit-Remaining") == "0" {
		reset, err := strconv.ParseInt(h.Get("X-RateLimit-Reset"), 10, 64)
		if err != nil {
			return time.Minute, true
		}
		now := time.Now
		if c.Now != nil {
			now = c.Now
		}
		d := time.Unix(reset, 0).Sub(now())
		if d < time.Second {
			d = time.Second
		}
		return d, true
	}
	return 0, false
}

// scrub removes the token from a message, as a last guard. No message is built
// from a header, so this should never change anything.
func scrub(s, token string) string {
	if token == "" {
		return s
	}
	return strings.ReplaceAll(s, token, "[redacted]")
}
