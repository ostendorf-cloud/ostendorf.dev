package content

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestParseArtefactSample(t *testing.T) {
	t.Parallel()
	for _, lang := range Languages {
		doc, err := ParseArtefact(sampleArtefacts(t)[lang], lang)
		if err != nil {
			t.Fatalf("%s: %v", lang, err)
		}
		if doc.Language != lang || len(doc.Projects) != 1 || doc.Projects[0].GithubRepo != "example/repo" {
			t.Errorf("%s: unexpected %+v", lang, doc.Projects)
		}
		if len(doc.Skills) != 2 || len(doc.Skills[1].Evidence) != 0 || doc.Skills[1].Evidence == nil {
			t.Errorf("%s: skills %+v", lang, doc.Skills)
		}
		if doc.Experience[1].End != nil || doc.Contact.Address == nil || (lang == "en" && doc.Contact.Address.Street != nil) {
			t.Errorf("%s: nulls not kept", lang)
		}
		if got := doc.Certificates[0].ValidUntil; got == nil || *got != "2026-01" {
			t.Errorf("%s: valid_until %v", lang, got)
		}
	}
}

// artefact builds a minimal artefact with one extra fragment spliced in.
func artefact(version, lang, extra string) []byte {
	return []byte(`{"schema_version":` + version + `,"language":"` + lang + `",` + extra + `
"experience":[],"education":[],"certificates":[],"skills":[],"projects":[],"references":[],"languages":[],
"contact":{"email":"a@example.com","address":null,"profiles":[]}}`)
}

func TestParseArtefactTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		data    []byte
		lang    string
		wantErr string
	}{
		{name: "minimal", data: artefact("1", "en", ""), lang: "en"},
		{name: "unknown schema version", data: artefact("2", "en", ""), lang: "en", wantErr: "unknown schema_version 2"},
		{name: "schema version zero", data: artefact("0", "en", ""), lang: "en", wantErr: "unknown schema_version 0"},
		{
			name: "future shape still gets the version message", lang: "en", wantErr: "unknown schema_version 3",
			data: []byte(`{"schema_version":3,"language":["en"],"experience":"changed"}`),
		},
		{name: "missing schema version", data: []byte(`{"language":"en"}`), lang: "en", wantErr: "schema_version is missing"},
		{name: "malformed JSON", data: []byte(`{"schema_version":`), lang: "en", wantErr: "not valid JSON"},
		{name: "empty input", data: nil, lang: "en", wantErr: "not valid JSON"},
		{name: "missing language", data: []byte(`{"schema_version":1}`), lang: "en", wantErr: `language is "", want "en"`},
		{name: "wrong language", data: artefact("1", "de", ""), lang: "en", wantErr: `language is "de", want "en"`},
		{
			name: "unknown top level keys ignored", lang: "en",
			data: artefact("1", "en", `"volunteering":[{"id":"v1"}],"awards":[],"patents":null,"future":{"x":1},`),
		},
		{
			name: "unknown nested keys ignored", lang: "en",
			data: []byte(`{"schema_version":1,"language":"en",
"experience":[{"id":"e","employer":"E","title":"T","start":"2020-01","end":null,"summary":null,"location":"X","employment_type":"full"}],
"education":[{"id":"d","institution":"I","degree":"D","start":"2015-10","end":null,"summary":null,"grade":"1.0"}],
"certificates":[],"skills":[],"projects":[],"references":[],"languages":[],
"contact":{"email":"a@example.com","address":null,"profiles":[]}}`),
		},
		{
			name: "valid_until null", lang: "en",
			data: []byte(`{"schema_version":1,"language":"en","certificates":[{"id":"c","name":"N","issuer":"I","issued":"2023-01","valid_until":null}]}`),
		},
		{name: "missing lists become empty", data: []byte(`{"schema_version":1,"language":"en"}`), lang: "en"},
		{
			name: "project without owner", lang: "en", wantErr: "not owner/name",
			data: []byte(`{"schema_version":1,"language":"en","projects":[{"id":"p","github_repo":"repo","description":"d"}]}`),
		},
		{
			name: "project path traversal", lang: "en", wantErr: "owner is not",
			data: []byte(`{"schema_version":1,"language":"en","projects":[{"id":"p","github_repo":"../x/y","description":"d"}]}`),
		},
		{
			name: "project dot dot as repository", lang: "en", wantErr: "repository name is not",
			data: []byte(`{"schema_version":1,"language":"en","projects":[{"id":"p","github_repo":"owner/..","description":"d"}]}`),
		},
		{
			name: "trailing data after the document", lang: "en", wantErr: "not valid JSON",
			data: append(artefact("1", "en", ""), []byte(` {"x":1}`)...),
		},
		{
			name: "duplicate project id", lang: "en", wantErr: "appears twice",
			data: []byte(`{"schema_version":1,"language":"en","projects":[{"id":"p","github_repo":"a/b","description":"d"},{"id":"p","github_repo":"a/c","description":"d"}]}`),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := ParseArtefact(tc.data, tc.lang)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if doc.Experience == nil || doc.Education == nil || doc.Certificates == nil || doc.Skills == nil ||
				doc.Projects == nil || doc.References == nil || doc.Languages == nil || doc.Contact.Profiles == nil {
				t.Error("a list is nil")
			}
		})
	}
}

func TestUnknownVersionMessageIsActionable(t *testing.T) {
	t.Parallel()
	_, err := ParseArtefact(artefact("2", "en", ""), "en")
	for _, want := range []string{"cv artefact en", "this build reads version 1 only", "pin an older cv release"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("error %v lacks %q", err, want)
		}
	}
}

func TestNullableValidUntilSurvives(t *testing.T) {
	t.Parallel()
	doc, err := ParseArtefact([]byte(`{"schema_version":1,"language":"en","certificates":[{"id":"c","name":"N","issuer":"I","issued":"2023-01","valid_until":null}]}`), "en")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Certificates[0].ValidUntil != nil {
		t.Error("null became a value")
	}
}

func TestLoadFromMissingDataset(t *testing.T) {
	t.Parallel()
	_, err := Load(fstest.MapFS{"generated/.gitkeep": {}})
	if err == nil || !strings.Contains(err.Error(), "task content:build") {
		t.Errorf("error %v", err)
	}
}
