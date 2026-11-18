package content

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// maxRedirects is how many redirects either client follows.
const maxRedirects = 5

// ReleaseRedirectHosts are the hosts a release asset download may be redirected
// to, besides the host the download started at. GitHub answers a release asset
// request with a redirect to its asset CDN. These names are the ones documented
// and seen in the past. They were not verified live for this change: if GitHub
// moves the CDN, the build fails with a refused redirect naming the host, and
// this list is the one place to extend after checking it.
var ReleaseRedirectHosts = []string{
	"github.com",
	"objects.githubusercontent.com",
	"release-assets.githubusercontent.com",
}

// errRedirectRefused marks a redirect the policy refused. It is final: a retry
// would be refused the same way.
var errRedirectRefused = errors.New("redirect refused")

// redirectPolicy returns a CheckRedirect that follows at most maxRedirects hops,
// only to https URLs on a host allowed returns true for, and that never carries
// an Authorization header to a host other than the one the request started at.
func redirectPolicy(allowed func(u *url.URL) bool) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		switch {
		case len(via) > maxRedirects:
			return fmt.Errorf("%w: more than %d redirects", errRedirectRefused, maxRedirects)
		case req.URL.Scheme != "https":
			return fmt.Errorf("%w: to a %s URL on %s, only https is followed", errRedirectRefused, req.URL.Scheme, req.URL.Host)
		case !allowed(req.URL):
			return fmt.Errorf("%w: to host %s, which is not allowed", errRedirectRefused, req.URL.Host)
		}
		if req.URL.Host != via[0].URL.Host {
			req.Header.Del("Authorization")
		}
		return nil
	}
}

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

// DigestEnv is the name of the cv-release.env value that pins lang's artefact.
func DigestEnv(lang string) string {
	return "CV_RELEASE_SHA256_" + strings.ToUpper(lang)
}

// CheckDigest verifies that raw, the artefact of lang, has the sha256 pinned in
// cv-release.env. A release tag is not immutable, so this pin is what makes a
// moved tag or a replaced asset fail the build instead of changing it silently.
// An empty pin is an error too. The message names the file and the tag and
// prints no more than the two digests.
func CheckDigest(lang, tag string, raw []byte, want string) error {
	name := DigestEnv(lang)
	if want == "" {
		return fmt.Errorf("cv-release.env: %s is empty: with %s pinned, set it to the sha256 of %s (sha256sum)", name, tag, ArtefactFileName(lang))
	}
	wantLower := strings.ToLower(want)
	if b, err := hex.DecodeString(wantLower); err != nil || len(b) != sha256.Size {
		return fmt.Errorf("cv-release.env: %s is not a sha256, 64 hexadecimal digits are expected", name)
	}
	sum := sha256.Sum256(raw)
	if got := hex.EncodeToString(sum[:]); got != wantLower {
		return fmt.Errorf("cv artefact %s of release %s does not match %s in cv-release.env: got %s, pinned %s",
			ArtefactFileName(lang), tag, name, got, wantLower)
	}
	return nil
}

// releaseURL builds the download URL of one asset. Every segment is escaped.
func releaseURL(base, tag, file string) (string, error) {
	owner, repo, _ := strings.Cut(CVRepository, "/")
	u, err := url.JoinPath(base, url.PathEscape(owner), url.PathEscape(repo), "releases", "download", url.PathEscape(tag), url.PathEscape(file))
	if err != nil {
		return "", fmt.Errorf("release url: %w", err)
	}
	return u, nil
}

// releaseRedirectAllowed reports whether a release asset download may follow a
// redirect to u: the host the download started at, or a name in
// ReleaseRedirectHosts, matched whole, never as a suffix.
func releaseRedirectAllowed(baseHost string) func(u *url.URL) bool {
	return func(u *url.URL) bool {
		if u.Host == baseHost {
			return true
		}
		for _, h := range ReleaseRedirectHosts {
			if u.Hostname() == h {
				return true
			}
		}
		return false
	}
}

// DownloadArtefacts fetches every language's artefact from the pinned release.
// The assets are public, so the request never carries a token: this function
// has none to send. Redirects are followed to https URLs on ReleaseRedirectHosts
// or on the host of base only. base is DefaultReleaseBase when empty.
func DownloadArtefacts(ctx context.Context, hc *http.Client, base, tag string) (map[string][]byte, error) {
	if err := ValidateTag(tag); err != nil {
		return nil, err
	}
	if base == "" {
		base = DefaultReleaseBase
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("release base: %w", err)
	}
	client := http.Client{Timeout: 30 * time.Second}
	if hc != nil {
		client = *hc
	}
	client.CheckRedirect = redirectPolicy(releaseRedirectAllowed(baseURL.Host))

	out := make(map[string][]byte, len(Languages))
	for _, lang := range Languages {
		u, err := releaseURL(base, tag, ArtefactFileName(lang))
		if err != nil {
			return nil, err
		}
		data, err := download(ctx, &client, u)
		if err != nil {
			return nil, err
		}
		out[lang] = data
	}
	return out, nil
}

func download(ctx context.Context, hc *http.Client, target string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", target, err)
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", target, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: status %d", target, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxArtefact+1))
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", target, err)
	}
	if len(data) > maxArtefact {
		return nil, fmt.Errorf("download %s: larger than %d bytes", target, maxArtefact)
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
