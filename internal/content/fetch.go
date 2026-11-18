package content

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// DefaultReleaseBase is where GitHub serves release assets from.
const DefaultReleaseBase = "https://github.com"

// CVRepository is the repository whose release carries the artefacts.
const CVRepository = "ostendorf-cloud/cv"

// maxArtefact caps one artefact download. The real ones are a few kilobytes.
const maxArtefact = 8 << 20

// tagPattern accepts a semantic version tag only. A moving reference such as
// latest, main or a branch name does not match, which is what keeps the pin a
// pin, and the pattern keeps the tag safe to put into a URL path.
var tagPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$`)

// ValidateTag reports whether tag names a release rather than a moving reference.
func ValidateTag(tag string) error {
	if tag == "" {
		return fmt.Errorf("no cv release is pinned: set CV_RELEASE_TAG in cv-release.env to a release tag such as v1.0.0")
	}
	if !tagPattern.MatchString(tag) {
		return fmt.Errorf("cv release tag %q is not a version tag such as v1.0.0: a moving reference is not a pin", tag)
	}
	return nil
}

// DownloadArtefacts fetches every language's artefact from the pinned release.
// The assets are public, so the request carries no token. base is
// DefaultReleaseBase when empty.
func DownloadArtefacts(ctx context.Context, hc *http.Client, base, tag string) (map[string][]byte, error) {
	if err := ValidateTag(tag); err != nil {
		return nil, err
	}
	if base == "" {
		base = DefaultReleaseBase
	}
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	out := make(map[string][]byte, len(Languages))
	for _, lang := range Languages {
		url := fmt.Sprintf("%s/%s/releases/download/%s/%s", strings.TrimRight(base, "/"), CVRepository, tag, ArtefactFileName(lang))
		data, err := download(ctx, hc, url)
		if err != nil {
			return nil, err
		}
		out[lang] = data
	}
	return out, nil
}

func download(ctx context.Context, hc *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: status %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxArtefact+1))
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	if len(data) > maxArtefact {
		return nil, fmt.Errorf("download %s: larger than %d bytes", url, maxArtefact)
	}
	return data, nil
}

// ReadArtefacts reads every language's artefact from a local directory, for an
// offline build and for the tests.
func ReadArtefacts(dir string) (map[string][]byte, error) {
	out := make(map[string][]byte, len(Languages))
	for _, lang := range Languages {
		data, err := os.ReadFile(filepath.Join(dir, ArtefactFileName(lang)))
		if err != nil {
			return nil, fmt.Errorf("read artefact: %w", err)
		}
		out[lang] = data
	}
	return out, nil
}
