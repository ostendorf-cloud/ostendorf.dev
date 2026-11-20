package webapp

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"

	"github.com/ostendorf-cloud/ostendorf.dev/internal/content"
	"github.com/ostendorf-cloud/ostendorf.dev/internal/routes"
)

// ProjectsSlug is the project index's slug, the same one routes.json carries.
// A project page lives one level below it.
const ProjectsSlug = "projects"

// projectIDPattern is what a project id must look like to be a path segment:
// lower case words joined by single hyphens. web/src/projects/ids.ts holds
// the same rule for the prerender script.
var projectIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ProjectRoutes returns one route per project of the dataset, sorted by id.
// The project pages are not in routes.json because their ids come from the
// generated dataset. Both languages must name the same projects, and every id
// must be a safe path segment, or the build is wrong and this fails.
func ProjectRoutes(d content.Dataset) ([]routes.Route, error) {
	var errs []error
	var ids []string
	sets := map[string][]string{}
	for _, lang := range content.Languages {
		doc, ok := d.CV[lang]
		if !ok {
			continue
		}
		var langIDs []string
		for _, p := range doc.Projects {
			if !projectIDPattern.MatchString(p.ID) {
				errs = append(errs, fmt.Errorf("project id %q (%s) is not a path segment: use lower case words joined by hyphens", p.ID, lang))
			}
			langIDs = append(langIDs, p.ID)
		}
		slices.Sort(langIDs)
		sets[lang] = langIDs
		ids = langIDs
	}
	for _, lang := range content.Languages {
		if !slices.Equal(sets[lang], ids) {
			errs = append(errs, fmt.Errorf("the %s and the other documents name different projects", lang))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("webapp: %w", err)
	}

	out := make([]routes.Route, 0, len(ids))
	for _, id := range ids {
		out = append(out, routes.Route{
			Name: "project:" + id,
			Slug: ProjectsSlug + "/" + id,
			File: ProjectsSlug + "/" + id + "/index.html",
		})
	}
	return out, nil
}

// withExtraRoutes returns a copy of the static slug to file map with the
// extra routes added. A slug that the table already has is an error: the
// extra route would silently replace a page.
func withExtraRoutes(static map[string]string, extra []routes.Route) (map[string]string, error) {
	merged := maps.Clone(static)
	for _, r := range extra {
		if _, taken := merged[r.Slug]; taken {
			return nil, fmt.Errorf("webapp: route %q repeats the slug %q", r.Name, r.Slug)
		}
		merged[r.Slug] = r.File
	}
	return merged, nil
}
