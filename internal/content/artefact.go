// Package content builds the dataset the site's pages read. At build time it
// fetches the cv repository's JSON release artefact, calls the GitHub API for
// every project the artefact names, merges both into one dataset and writes it
// to a file the binary embeds. Nothing in this package runs at request time.
package content

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

// SupportedSchemaVersion is the only cv artefact schema version this package
// reads. The cv repository raises its version on any change a consumer reading
// the old shape would get wrong, so an unknown version fails the build instead
// of being guessed at.
const SupportedSchemaVersion = 1

// Languages is every language the cv repository publishes, one artefact each.
var Languages = []string{"de", "en"}

// ArtefactFileName is the release asset name of the artefact for lang, for
// example cv.en.json. It is part of the contract with the cv repository.
func ArtefactFileName(lang string) string {
	return "cv." + lang + ".json"
}

// repoPattern is the owner/name form of a public GitHub repository. It is also
// what keeps a path segment from ever carrying anything but a repository name
// into a request URL.
var repoPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// Document is one language's CV data. The artefact parser fills every field
// except Project.GitHub, and the merge fills that one. Unknown keys in an
// artefact are ignored: the cv repository adds keys a consumer can ignore
// without raising its schema version, and this type carries only what a page
// of this site needs. The sections cv appends after contact (volunteering,
// awards, exam_results, courses, publications, patents, organisations) and the
// optional grade, location and employment_type keys are dropped here until a
// page needs one of them.
type Document struct {
	Language     string        `json:"language"`
	Experience   []Experience  `json:"experience"`
	Education    []Education   `json:"education"`
	Certificates []Certificate `json:"certificates"`
	Skills       []Skill       `json:"skills"`
	Projects     []Project     `json:"projects"`
	References   []Reference   `json:"references"`
	Languages    []Proficiency `json:"languages"`
	Contact      Contact       `json:"contact"`
}

// Experience is one employment.
type Experience struct {
	ID       string  `json:"id"`
	Employer string  `json:"employer"`
	Title    string  `json:"title"`
	Start    string  `json:"start"`
	End      *string `json:"end"`
	Summary  *string `json:"summary"`
}

// Education is one course of study.
type Education struct {
	ID          string  `json:"id"`
	Institution string  `json:"institution"`
	Degree      string  `json:"degree"`
	Start       string  `json:"start"`
	End         *string `json:"end"`
	Summary     *string `json:"summary"`
}

// Certificate is one certificate. ValidUntil is null for one that does not
// expire.
type Certificate struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Issuer     string  `json:"issuer"`
	Issued     string  `json:"issued"`
	ValidUntil *string `json:"valid_until"`
}

// Evidence points from a skill to the entry that backs it.
type Evidence struct {
	Type string `json:"type"`
	Ref  string `json:"ref"`
}

// Skill is one skill or tool with the evidence behind it.
type Skill struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Level    string     `json:"level"`
	Evidence []Evidence `json:"evidence"`
}

// Project is one project: the hand written description comes from the cv
// artefact, GitHub from the GitHub API. GitHub is nil only between parsing an
// artefact and merging.
type Project struct {
	ID          string `json:"id"`
	GithubRepo  string `json:"github_repo"`
	Description string `json:"description"`
	GitHub      *Repo  `json:"github"`
}

// Reference is one person who vouches for the author.
type Reference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

// Proficiency is one spoken language and the level it is spoken at.
type Proficiency struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Level string `json:"level"`
}

// Address is the postal address, every part optional.
type Address struct {
	Street *string `json:"street"`
	City   *string `json:"city"`
}

// Profile is one profile on another platform.
type Profile struct {
	ID       string `json:"id"`
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

// Contact is how to reach the author.
type Contact struct {
	Email    string    `json:"email"`
	Address  *Address  `json:"address"`
	Profiles []Profile `json:"profiles"`
}

// artefactDocument is the artefact's shape on the wire: Document plus the
// schema version, which the dataset carries once, not per language.
type artefactDocument struct {
	SchemaVersion int `json:"schema_version"`
	Document
}

// ParseArtefact reads the artefact of lang. It checks the schema version before
// it reads anything else, so a document of a future shape gets a clear message
// instead of a decoding error about one of its fields. It ignores unknown keys.
// A list that is missing or null becomes an empty list.
func ParseArtefact(data []byte, lang string) (Document, error) {
	var head struct {
		SchemaVersion *int `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return Document{}, fmt.Errorf("cv artefact %s: not valid JSON: %w", lang, err)
	}
	if head.SchemaVersion == nil {
		return Document{}, fmt.Errorf("cv artefact %s: schema_version is missing", lang)
	}
	if *head.SchemaVersion != SupportedSchemaVersion {
		return Document{}, fmt.Errorf(
			"cv artefact %s: unknown schema_version %d, this build reads version %d only: "+
				"update internal/content for the new version, or pin an older cv release",
			lang, *head.SchemaVersion, SupportedSchemaVersion)
	}

	var doc artefactDocument
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&doc); err != nil {
		return Document{}, fmt.Errorf("cv artefact %s: %w", lang, err)
	}
	if doc.Language != lang {
		return Document{}, fmt.Errorf("cv artefact %s: language is %q, want %q", lang, doc.Language, lang)
	}
	d := doc.Document
	d.normalize()
	if err := d.validate(); err != nil {
		return Document{}, fmt.Errorf("cv artefact %s: %w", lang, err)
	}
	return d, nil
}

// normalize turns a missing or null list into an empty one, so a page never
// has to tell the two apart.
func (d *Document) normalize() {
	if d.Experience == nil {
		d.Experience = []Experience{}
	}
	if d.Education == nil {
		d.Education = []Education{}
	}
	if d.Certificates == nil {
		d.Certificates = []Certificate{}
	}
	if d.Skills == nil {
		d.Skills = []Skill{}
	}
	for i := range d.Skills {
		if d.Skills[i].Evidence == nil {
			d.Skills[i].Evidence = []Evidence{}
		}
	}
	if d.Projects == nil {
		d.Projects = []Project{}
	}
	if d.References == nil {
		d.References = []Reference{}
	}
	if d.Languages == nil {
		d.Languages = []Proficiency{}
	}
	if d.Contact.Profiles == nil {
		d.Contact.Profiles = []Profile{}
	}
}

// validate checks what the merge relies on: a project names a public
// repository in owner/name form, and project ids do not repeat.
func (d *Document) validate() error {
	var errs []error
	seen := map[string]bool{}
	for i, p := range d.Projects {
		if p.ID == "" {
			errs = append(errs, fmt.Errorf("projects[%d]: id is empty", i))
		} else if seen[p.ID] {
			errs = append(errs, fmt.Errorf("projects[%d]: id %q appears twice", i, p.ID))
		}
		seen[p.ID] = true
		if !repoPattern.MatchString(p.GithubRepo) {
			errs = append(errs, fmt.Errorf("projects[%d] %q: github_repo %q is not owner/name", i, p.ID, p.GithubRepo))
		}
	}
	return errors.Join(errs...)
}
