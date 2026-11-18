package content

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"time"
)

// DatasetSchemaVersion is the version of the merged dataset the pages read. It
// is this repository's own contract, separate from the cv artefact's.
const DatasetSchemaVersion = 1

// DatasetFile is the generated file's name, inside the generated directory.
const DatasetFile = "dataset.json"

// Dataset is the merged result the binary embeds. The same inputs and the same
// clock give the same bytes: every map is written in key order and every list
// has a fixed order.
type Dataset struct {
	SchemaVersion int `json:"schema_version"`
	// GeneratedAt is when this file was written, RFC 3339 in UTC. It is the one
	// value that comes from the clock.
	GeneratedAt string `json:"generated_at"`
	Source      Source `json:"source"`
	// CV holds one document per language, keyed by language.
	CV map[string]Document `json:"cv"`
}

// Source says where the data came from.
type Source struct {
	CV     CVSource     `json:"cv"`
	GitHub GitHubSource `json:"github"`
}

// CVSource names the pinned cv release. SHA256 holds the digest of each
// artefact as it was read, so the file records exactly what went into it.
type CVSource struct {
	Repository    string            `json:"repository"`
	Release       string            `json:"release"`
	SchemaVersion int               `json:"schema_version"`
	SHA256        map[string]string `json:"sha256"`
}

// GitHubSource says when the GitHub data was fetched and whether it is the
// last known good data instead of a fresh fetch.
type GitHubSource struct {
	FetchedAt   string  `json:"fetched_at"`
	Stale       bool    `json:"stale"`
	StaleReason *string `json:"stale_reason"`
}

// Encode writes the dataset as two space indented JSON with a trailing
// newline, not HTML escaped.
func (d Dataset) Encode() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(d); err != nil {
		return nil, fmt.Errorf("encode dataset: %w", err)
	}
	return buf.Bytes(), nil
}

// ParseDataset reads a dataset strictly: an unknown key is an error, so the
// writer and the reader cannot drift apart unnoticed. It also runs Validate.
func ParseDataset(data []byte) (Dataset, error) {
	var d Dataset
	if err := decodeOne(data, &d, true); err != nil {
		return Dataset{}, fmt.Errorf("dataset: %w", err)
	}
	if d.SchemaVersion != DatasetSchemaVersion {
		return Dataset{}, fmt.Errorf("dataset: schema_version %d, this build reads %d", d.SchemaVersion, DatasetSchemaVersion)
	}
	if err := d.Validate(); err != nil {
		return Dataset{}, err
	}
	return d, nil
}

// decodeOne decodes exactly one JSON value into v. Anything after it, a second
// object or stray text, is an error.
func decodeOne(data []byte, v any, strict bool) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	if strict {
		dec.DisallowUnknownFields()
	}
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("unexpected data after the JSON document")
	}
	return nil
}

// Validate checks the dataset's own shape: one document per language, every
// project merged with its GitHub data, and a complete source block. This is the
// dataset's schema check. It runs in the build step on the bytes just written,
// so a dataset that fails it never reaches the binary.
func (d Dataset) Validate() error {
	var errs []error
	for _, lang := range Languages {
		doc, ok := d.CV[lang]
		if !ok {
			errs = append(errs, fmt.Errorf("dataset: no document for language %q", lang))
			continue
		}
		if doc.Language != lang {
			errs = append(errs, fmt.Errorf("dataset: document under %q has language %q", lang, doc.Language))
		}
		for _, p := range doc.Projects {
			if p.GitHub == nil {
				errs = append(errs, fmt.Errorf("dataset: %s project %q has no GitHub data", lang, p.ID))
			}
		}
	}
	if len(d.CV) != len(Languages) {
		errs = append(errs, fmt.Errorf("dataset: %d documents, want %d", len(d.CV), len(Languages)))
	}
	if _, err := time.Parse(time.RFC3339, d.GeneratedAt); err != nil {
		errs = append(errs, fmt.Errorf("dataset: generated_at: %w", err))
	}
	if _, err := time.Parse(time.RFC3339, d.Source.GitHub.FetchedAt); err != nil {
		errs = append(errs, fmt.Errorf("dataset: source.github.fetched_at: %w", err))
	}
	if d.Source.GitHub.Stale != (d.Source.GitHub.StaleReason != nil) {
		errs = append(errs, errors.New("dataset: source.github.stale and stale_reason disagree"))
	}
	if d.Source.CV.Release == "" || d.Source.CV.SchemaVersion != SupportedSchemaVersion {
		errs = append(errs, errors.New("dataset: source.cv lacks a release or carries an unknown schema_version"))
	}
	return errors.Join(errs...)
}

// Load reads the dataset out of a filesystem that holds generated/dataset.json.
func Load(fsys fs.FS) (Dataset, error) {
	data, err := fs.ReadFile(fsys, "generated/"+DatasetFile)
	if err != nil {
		return Dataset{}, fmt.Errorf("content: the embedded dataset is missing, run task content:build: %w", err)
	}
	return ParseDataset(data)
}
