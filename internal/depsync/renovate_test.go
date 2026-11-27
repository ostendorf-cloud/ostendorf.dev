package depsync

import (
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// renovateConfigPath is the repository's Renovate configuration, relative to this package.
const renovateConfigPath = "../../renovate.json"

// govuePresetName is the shared preset that regenerates web/pnpm-lock.yaml for a govue pull request.
const govuePresetName = "github>ostendorf-cloud/workflows:govue-lockfile#v0"

// renovateConfig holds the parts of renovate.json these checks read.
type renovateConfig struct {
	Extends        []string `json:"extends"`
	CustomManagers []struct {
		CustomType          string   `json:"customType"`
		ManagerFilePatterns []string `json:"managerFilePatterns"`
		MatchStrings        []string `json:"matchStrings"`
		DepNameTemplate     string   `json:"depNameTemplate"`
		DatasourceTemplate  string   `json:"datasourceTemplate"`
	} `json:"customManagers"`
	PackageRules []struct {
		MatchPackageNames []string          `json:"matchPackageNames"`
		Description       string            `json:"description"`
		AllowedVersions   string            `json:"allowedVersions"`
		MatchManagers     []string          `json:"matchManagers"`
		GroupName         string            `json:"groupName"`
		Constraints       map[string]string `json:"constraints"`
	} `json:"packageRules"`
}

func readRenovateConfig(t *testing.T) renovateConfig {
	t.Helper()

	raw, err := os.ReadFile(renovateConfigPath)
	if err != nil {
		t.Fatalf("read renovate.json: %v", err)
	}

	var cfg renovateConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse renovate.json: %v", err)
	}
	return cfg
}

// TestRenovateExtendsGovueLockfilePreset fails when renovate.json drops the preset. Without
// it a govue pull request carries a stale web/pnpm-lock.yaml and fails check:govue-version.
func TestRenovateExtendsGovueLockfilePreset(t *testing.T) {
	t.Parallel()

	if !slices.Contains(readRenovateConfig(t).Extends, govuePresetName) {
		t.Fatalf("renovate.json does not extend %q", govuePresetName)
	}
}

// TestRenovateGroupsGovueHalves fails when renovate.json no longer puts the Go module and
// the npm package of govue into one group. Without the group Renovate raises the halves in
// separate pull requests, and each fails check:govue-version.
func TestRenovateGroupsGovueHalves(t *testing.T) {
	t.Parallel()

	cfg := readRenovateConfig(t)

	var groups []string
	for _, rule := range cfg.PackageRules {
		if rule.GroupName == "" {
			continue
		}
		if slices.Contains(rule.MatchPackageNames, goModulePath) &&
			slices.Contains(rule.MatchPackageNames, "ostendorf-cloud/govue") {
			groups = append(groups, rule.GroupName)
		}
	}

	if len(groups) != 1 {
		t.Fatalf("want exactly one packageRules group naming both %q and %q, got %d: %v",
			goModulePath, "ostendorf-cloud/govue", len(groups), groups)
	}
}

// TestRenovateManagerReadsGovueSpecifier fails when the custom manager's expression no
// longer extracts the tag from web/package.json's real specifier. Renovate's own npm
// manager skips that specifier ("unversioned-reference") because of the "&path:/web"
// suffix, so this manager is the only way the npm half is ever updated.
func TestRenovateManagerReadsGovueSpecifier(t *testing.T) {
	t.Parallel()

	cfg := readRenovateConfig(t)

	raw, err := os.ReadFile(packageJSONPath)
	if err != nil {
		t.Fatalf("read web/package.json: %v", err)
	}
	wantTag, err := packageJSONGovueTag(t)
	if err != nil {
		t.Fatalf("resolve the tag in web/package.json: %v", err)
	}

	for _, manager := range cfg.CustomManagers {
		if manager.CustomType != "regex" || manager.DepNameTemplate != "ostendorf-cloud/govue" {
			continue
		}
		if manager.DatasourceTemplate != "github-tags" {
			t.Fatalf("datasourceTemplate = %q, want github-tags", manager.DatasourceTemplate)
		}
		if !slices.Contains(manager.ManagerFilePatterns, "/^web/package\\.json$/") {
			t.Fatalf("managerFilePatterns = %v, want web/package.json", manager.ManagerFilePatterns)
		}
		if len(manager.MatchStrings) != 1 {
			t.Fatalf("want one matchStrings entry, got %d", len(manager.MatchStrings))
		}

		re := regexp.MustCompile(manager.MatchStrings[0])
		group := re.SubexpIndex("currentValue")
		if group < 0 {
			t.Fatalf("matchStrings %q has no named group currentValue", manager.MatchStrings[0])
		}
		match := re.FindSubmatch(raw)
		if match == nil {
			t.Fatalf("matchStrings %q finds nothing in web/package.json", manager.MatchStrings[0])
		}
		got := string(match[group])
		if got != wantTag {
			t.Fatalf("custom manager extracts tag %q, web/package.json pins %q", got, wantTag)
		}
		return
	}

	t.Fatal("renovate.json has no regex custom manager for ostendorf-cloud/govue")
}

// TestRenovateGovuePnpmConstraintMatchesMise fails when the govue rule's constraints.pnpm
// differs from the pnpm pin in mise.toml. The post upgrade lock file command runs with that
// constraint, so a lock file written by another pnpm fails the frozen install later.
func TestRenovateGovuePnpmConstraintMatchesMise(t *testing.T) {
	t.Parallel()

	mise, err := os.ReadFile("../../mise.toml")
	if err != nil {
		t.Fatal(err)
	}
	pin := regexp.MustCompile(`(?m)^pnpm = "([^"]+)"`).FindStringSubmatch(string(mise))
	if pin == nil {
		t.Fatal(`mise.toml has no 'pnpm = "<version>"' line`)
	}

	for _, rule := range readRenovateConfig(t).PackageRules {
		if rule.GroupName != "govue" {
			continue
		}
		got, ok := rule.Constraints["pnpm"]
		if !ok {
			t.Fatal("the govue rule has no constraints.pnpm")
		}
		if got != pin[1] {
			t.Fatalf("constraints.pnpm is %s, mise.toml pins %s: they move together", got, pin[1])
		}
		return
	}
	t.Fatal("renovate.json has no package rule with groupName govue")
}

// TestRenovateGroupsPnpmPins fails when renovate.json has no rule that groups the mise.toml
// pin and the govue rule's constraints.pnpm as one pnpm update. Without the group each bump
// of one pin arrives alone and fails TestRenovateGovuePnpmConstraintMatchesMise.
func TestRenovateGroupsPnpmPins(t *testing.T) {
	t.Parallel()

	for _, rule := range readRenovateConfig(t).PackageRules {
		if !slices.Contains(rule.MatchPackageNames, "pnpm") {
			continue
		}
		for _, manager := range []string{"mise", "custom.regex", "renovate-config"} {
			if !slices.Contains(rule.MatchManagers, manager) {
				t.Fatalf("the pnpm rule matches managers %v, want %q among them", rule.MatchManagers, manager)
			}
		}
		if rule.GroupName != "pnpm" {
			t.Fatalf("the pnpm rule has groupName %q, want pnpm", rule.GroupName)
		}
		return
	}
	t.Fatal(`renovate.json has no package rule with matchPackageNames "pnpm"`)
}

// TestRenovateManagerReadsDockerfilePnpm fails when the regex manager for the Dockerfile no
// longer extracts the pnpm version from the real Dockerfile. The manager is the only way
// Renovate sees PNPM_VERSION, so a broken expression leaves the image pin behind mise.toml.
func TestRenovateManagerReadsDockerfilePnpm(t *testing.T) {
	t.Parallel()

	docker, err := os.ReadFile("../../Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	mise, err := os.ReadFile("../../mise.toml")
	if err != nil {
		t.Fatal(err)
	}
	pin := misePnpm.FindStringSubmatch(string(mise))
	if pin == nil {
		t.Fatal(`mise.toml has no 'pnpm = "<version>"' line`)
	}

	for _, manager := range readRenovateConfig(t).CustomManagers {
		if manager.CustomType != "regex" || !slices.Contains(manager.ManagerFilePatterns, "/^Dockerfile$/") {
			continue
		}
		if len(manager.MatchStrings) != 1 {
			t.Fatalf("want one matchStrings entry, got %d", len(manager.MatchStrings))
		}

		re, err := regexp.Compile(manager.MatchStrings[0])
		if err != nil {
			t.Fatalf("matchStrings %q does not compile: %v", manager.MatchStrings[0], err)
		}
		match := re.FindStringSubmatch(string(docker))
		if match == nil {
			t.Fatalf("matchStrings %q finds nothing in the Dockerfile", manager.MatchStrings[0])
		}
		want := map[string]string{"datasource": "npm", "depName": "pnpm", "currentValue": pin[1]}
		for name, value := range want {
			idx := re.SubexpIndex(name)
			if idx < 0 {
				t.Fatalf("matchStrings %q has no named group %s", manager.MatchStrings[0], name)
			}
			if match[idx] != value {
				t.Fatalf("group %s extracts %q, want %q", name, match[idx], value)
			}
		}
		return
	}

	t.Fatal("renovate.json has no regex custom manager for the Dockerfile")
}

// TestRenovateHoldsTypescriptBelowSeven fails when renovate.json has no rule that keeps
// typescript below 7, or when the rule lost its range or its removal condition. Without the
// rule Renovate raises TypeScript 7, which typescript-eslint and vue-tsc cannot load yet.
func TestRenovateHoldsTypescriptBelowSeven(t *testing.T) {
	t.Parallel()

	for _, rule := range readRenovateConfig(t).PackageRules {
		if !slices.Contains(rule.MatchPackageNames, "typescript") {
			continue
		}
		if rule.AllowedVersions != "<7" {
			t.Fatalf("the typescript rule has allowedVersions %q, want %q", rule.AllowedVersions, "<7")
		}
		if !strings.Contains(rule.Description, "Remove this rule when") {
			t.Fatalf("the typescript rule description names no removal condition: %q", rule.Description)
		}
		return
	}
	t.Fatal(`renovate.json has no package rule with matchPackageNames "typescript"`)
}
