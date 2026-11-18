package content

import "embed"

// generatedFS holds the dataset "task content:build" writes. The directory is
// committed with a .gitkeep and nothing else, and "all:" makes that dotfile
// count, so the pattern matches on a clean checkout and go build and go test
// work without the generated file. dataset.json itself is gitignored, never
// committed. Embedded reports its absence, and the Dockerfile refuses to build
// an image without it.
//
//go:embed all:generated
var generatedFS embed.FS

// Embedded returns the dataset compiled into the binary.
func Embedded() (Dataset, error) {
	return loadFrom(generatedFS)
}
