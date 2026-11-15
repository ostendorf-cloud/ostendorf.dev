package main

import "embed"

// distFS is the built SPA, written by "task web:build" (vite build, with
// outDir pointed at this directory). embed resolves relative to this file,
// which is why the directory sits under cmd/ rather than internal/.
//
//go:embed dist
var distFS embed.FS

// prerenderedFS is the static HTML for every route, written by
// "task prerender" after dist exists.
//
//go:embed prerendered
var prerenderedFS embed.FS
