package main

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/ostendorf-cloud/ostendorf.dev/internal/content"
)

// TestStartRefusesWithoutTheDataset proves a binary built without
// "task content:build" does not start, and says what to run. main prints the
// error and exits 1.
func TestStartRefusesWithoutTheDataset(t *testing.T) {
	// Not parallel: run reads the process environment.
	err := runWith(func() (content.Dataset, error) {
		return content.Load(fstest.MapFS{"generated/.gitkeep": {}})
	})
	if err == nil || !strings.Contains(err.Error(), "task content:build") {
		t.Fatalf("error %v", err)
	}
}
