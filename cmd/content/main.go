// Command content is the build step that writes the dataset the website binary
// embeds. It runs in CI and on a developer machine, never in the deployed
// service: the binary holds no GitHub token and makes no call.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/ostendorf-cloud/govue/logging"

	"github.com/ostendorf-cloud/ostendorf.dev/internal/content"
)

// sleepFn waits between retries. Nil is the real wait. A test replaces it so a
// failing GitHub run does not sit through the backoff.
var sleepFn func(context.Context, time.Duration) error

const defaultOut = "internal/content/generated/" + content.DatasetFile

func main() {
	os.Exit(realMain())
}

func realMain() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Getenv, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "content:", err)
		return 1
	}
	return 0
}

// run is main without the process around it, so a test can drive it.
//
// Environment: CV_RELEASE_TAG is the pinned cv release, and
// CV_RELEASE_SHA256_DE and CV_RELEASE_SHA256_EN are the digests of its two
// artefacts, all three from cv-release.env. GITHUB_TOKEN is read
// when set and is optional, since public data needs no scope. SOURCE_DATE_EPOCH
// fixes the clock for a reproducible build.
func run(ctx context.Context, args []string, getenv func(string) string, stderr io.Writer) error {
	fs := flag.NewFlagSet("content", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", defaultOut, "path of the dataset to write")
	artefactDir := fs.String("artefact-dir", "", "read cv.<lang>.json from this directory instead of downloading the release")
	cvBase := fs.String("cv-base", content.DefaultReleaseBase, "base URL release assets are downloaded from")
	githubAPI := fs.String("github-api", content.DefaultGitHubAPI, "base URL of the GitHub REST API")
	previous := fs.String("previous", "", "an earlier dataset, the last known good one")
	allowStale := fs.Bool("allow-stale", false, "on a failed GitHub fetch, ship the data of -previous marked stale")
	if err := fs.Parse(args); err != nil {
		return err
	}

	logger := logging.New(stderr, slog.LevelInfo)
	now := time.Now
	if v := getenv("SOURCE_DATE_EPOCH"); v != "" {
		secs, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("SOURCE_DATE_EPOCH: %w", err)
		}
		now = func() time.Time { return time.Unix(secs, 0) }
	}

	local := *artefactDir != ""
	tag := getenv("CV_RELEASE_TAG")
	digests := map[string]string{}
	for _, lang := range content.Languages {
		digests[lang] = getenv(content.DigestEnv(lang))
	}
	if !local {
		if err := content.ValidateTag(tag); err != nil {
			return err
		}
	}

	var artefacts map[string][]byte
	var err error
	if local {
		artefacts, err = content.ReadArtefacts(*artefactDir)
	} else {
		artefacts, err = content.DownloadArtefacts(ctx, nil, *cvBase, tag)
	}
	if err != nil {
		return err
	}

	var prev []byte
	if *previous != "" {
		prev, err = os.ReadFile(*previous)
		// A missing file is "no previous dataset" when the fallback is allowed:
		// the first build has none, and Generate then fails on the GitHub error.
		missingOK := *allowStale && errors.Is(err, os.ErrNotExist)
		if err != nil && !missingOK {
			return fmt.Errorf("read previous dataset: %w", err)
		}
	}

	client := &content.Client{
		BaseURL: *githubAPI,
		HTTP:    &http.Client{Timeout: 15 * time.Second},
		Token:   getenv("GITHUB_TOKEN"),
		Sleep:   sleepFn,
		Log:     logger,
	}
	data, err := content.Generate(ctx, content.Options{
		Tag:        tag,
		Digests:    digests,
		Local:      local,
		Artefacts:  artefacts,
		GitHub:     client,
		Previous:   prev,
		AllowStale: *allowStale,
		Now:        now,
		Log:        logger,
	})
	if err != nil {
		return err
	}
	return writeFile(*out, data)
}

// writeFile replaces path in one step, so a failed run never leaves half a file.
func writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".dataset-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
