package content

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"
)

// RepoSource fetches repository data. *Client implements it.
type RepoSource interface {
	FetchRepos(ctx context.Context, names []string) (map[string]*Repo, error)
}

// DefaultMaxStaleAge is how old the last known good GitHub data may be before
// it is refused as well.
const DefaultMaxStaleAge = 14 * 24 * time.Hour

// DefaultGitHubTimeout bounds the whole GitHub phase of one build, so a slow or
// rate limited API ends as a clear error instead of a CI job timeout.
const DefaultGitHubTimeout = 3 * time.Minute

// Options is everything Generate needs. Nothing in it reads the environment or
// the network by itself.
type Options struct {
	// Tag is the pinned cv release, recorded in the dataset.
	Tag string
	// Digests is the sha256 pinned in cv-release.env for each language's
	// artefact. A mismatch or an empty value fails the build.
	Digests map[string]string
	// Local marks artefacts read from a local directory, not from the pinned
	// release. No tag and no digest is checked, and the dataset records the
	// release as "local" so its provenance is not misleading.
	Local bool
	// Artefacts is the raw artefact per language.
	Artefacts map[string][]byte
	// GitHubTimeout bounds the whole GitHub phase, every request and every wait
	// together, DefaultGitHubTimeout when zero.
	GitHubTimeout time.Duration
	// GitHub fetches the repository data.
	GitHub RepoSource
	// Previous is an earlier dataset, the last known good one. It is read only
	// when GitHub fails and AllowStale is set.
	Previous []byte
	// AllowStale lets a failed GitHub fetch fall back to Previous. Off by default:
	// a failed fetch fails the build.
	AllowStale bool
	// MaxStaleAge bounds Previous's age, DefaultMaxStaleAge when zero.
	MaxStaleAge time.Duration
	// Now is the clock, time.Now when nil. A fixed clock makes the output
	// reproducible.
	Now func() time.Time
	// Log receives the stale warning.
	Log *slog.Logger
}

// Generate parses the artefacts, fetches the GitHub data, merges both and
// returns the encoded dataset.
//
// Failure policy: an artefact that fails to parse or carries an unknown schema
// version always fails. A failed GitHub fetch fails too, unless AllowStale is
// set and Previous holds fresh enough data for every repository: then the
// dataset carries that data with stale set and the reason. The fallback is all
// or nothing, so a partial dataset never comes out.
func Generate(ctx context.Context, o Options) ([]byte, error) {
	release := o.Tag
	if o.Local {
		release = "local"
	} else if err := ValidateTag(o.Tag); err != nil {
		return nil, err
	}
	now := time.Now
	if o.Now != nil {
		now = o.Now
	}
	log := o.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}

	docs := make(map[string]Document, len(Languages))
	digests := make(map[string]string, len(Languages))
	var errs []error
	for _, lang := range Languages {
		raw, ok := o.Artefacts[lang]
		if !ok {
			errs = append(errs, fmt.Errorf("cv artefact %s: missing", lang))
			continue
		}
		if !o.Local {
			if err := CheckDigest(lang, o.Tag, raw, o.Digests[lang]); err != nil {
				errs = append(errs, err)
				continue
			}
		}
		doc, err := ParseArtefact(raw, lang)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		docs[lang] = doc
		sum := sha256.Sum256(raw)
		digests[lang] = hex.EncodeToString(sum[:])
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	names := repoNames(docs)
	fetchedAt := now().UTC()
	timeout := o.GitHubTimeout
	if timeout <= 0 {
		timeout = DefaultGitHubTimeout
	}
	fctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	repos, err := o.GitHub.FetchRepos(fctx, names)
	if err != nil && ctx.Err() == nil && fctx.Err() != nil {
		err = fmt.Errorf("the GitHub phase exceeded its overall deadline of %s: %w", timeout, err)
	}
	gh := GitHubSource{FetchedAt: fetchedAt.Format(time.RFC3339)}
	if err != nil {
		if !o.AllowStale {
			return nil, err
		}
		prev, prevFetched, perr := previousRepos(o.Previous, names, now(), o.MaxStaleAge)
		if perr != nil {
			return nil, fmt.Errorf("%w; no usable last known good data: %w", err, perr)
		}
		reason := "github fetch failed: " + err.Error()
		log.Warn("shipping stale GitHub data", "fetched_at", prevFetched.Format(time.RFC3339), "reason", err.Error())
		repos = prev
		gh = GitHubSource{FetchedAt: prevFetched.Format(time.RFC3339), Stale: true, StaleReason: &reason}
	}

	for lang, doc := range docs {
		merged := make([]Project, len(doc.Projects))
		for i, p := range doc.Projects {
			r, ok := repos[p.GithubRepo]
			if !ok {
				return nil, fmt.Errorf("merge: no GitHub data for %s", p.GithubRepo)
			}
			p.GitHub = r
			merged[i] = p
		}
		doc.Projects = merged
		docs[lang] = doc
	}

	ds := Dataset{
		SchemaVersion: DatasetSchemaVersion,
		GeneratedAt:   fetchedAt.Format(time.RFC3339),
		Source: Source{
			CV: CVSource{
				Repository:    CVRepository,
				Release:       release,
				SchemaVersion: SupportedSchemaVersion,
				SHA256:        digests,
			},
			GitHub: gh,
		},
		CV: docs,
	}
	out, err := ds.Encode()
	if err != nil {
		return nil, err
	}
	// The dataset's own check, on the bytes that will be embedded.
	if _, err := ParseDataset(out); err != nil {
		return nil, fmt.Errorf("generated dataset fails its own check: %w", err)
	}
	return out, nil
}

// repoNames is the sorted, deduplicated set of repositories the artefacts name.
// It is the only list that is ever requested from GitHub.
func repoNames(docs map[string]Document) []string {
	set := map[string]bool{}
	for _, d := range docs {
		for _, p := range d.Projects {
			set[p.GithubRepo] = true
		}
	}
	names := make([]string, 0, len(set))
	for n := range set {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// previousRepos extracts the GitHub data of an earlier dataset. It fails
// unless the data is fresh enough and covers every wanted repository.
func previousRepos(prev []byte, want []string, now time.Time, maxAge time.Duration) (map[string]*Repo, time.Time, error) {
	if len(prev) == 0 {
		return nil, time.Time{}, errors.New("no previous dataset given")
	}
	ds, err := ParseDataset(prev)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("previous dataset: %w", err)
	}
	fetched, err := time.Parse(time.RFC3339, ds.Source.GitHub.FetchedAt)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("previous dataset: %w", err)
	}
	if maxAge <= 0 {
		maxAge = DefaultMaxStaleAge
	}
	age := now.Sub(fetched)
	if age < 0 {
		return nil, time.Time{}, fmt.Errorf("previous dataset's GitHub data claims to be fetched at %s, which is in the future", fetched.Format(time.RFC3339))
	}
	if age > maxAge {
		return nil, time.Time{}, fmt.Errorf("previous dataset's GitHub data is %s old, the limit is %s", age.Round(time.Hour), maxAge)
	}
	have := map[string]*Repo{}
	for _, d := range ds.CV {
		for _, p := range d.Projects {
			have[p.GithubRepo] = p.GitHub
		}
	}
	for _, n := range want {
		if have[n] == nil {
			return nil, time.Time{}, fmt.Errorf("previous dataset has no data for %s", n)
		}
	}
	return have, fetched.UTC(), nil
}
