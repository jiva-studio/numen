// Package direction is the check `go/direction`, run on the tree with the planted violations that prove it.
package direction

import (
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// getRefusedImports defines the complete rule, so the test calls it directly.
// A walk that reads the tree but rejects nothing would pass, so these cases pin
// the forbidden edges.
func TestWhatTheDirectionRulesRefuse(t *testing.T) {
	for _, one := range []struct {
		pkg, dep  string
		isRefused bool
	}{
		// The core is compiled from none of the three, at any remove.
		{"usecase/note", load.Module + "adapter/window/editor", true},
		{"usecase/note", load.Module + "adapter/window/theme", true},
		{"usecase/note", load.Module + "container", true},
		{"usecase/note", load.Wire + "/gen/numen/v1", true},
		{"text", load.Module + "internal/adapter/filesystem", true},

		// The composition root is built from the adapters and answers for what
		// it names. The schema is the one thing it may not name.
		{"container", load.Wire + "/gen/numen/v1", true},
		{"container", load.Module + "adapter/index", false},
		{"container", load.Module + "adapter/window/theme", false},

		// does not.
		{"internal/adapterlib/wire", load.Wire + "/gen/numen/v1", false},
		{"internal/adapterlib/onnxruntime", load.Wire + "/gen/numen/v1", false},
		{"internal/query", load.Wire + "/gen/numen/v1", true},

		// An adapter, and what only a test builds, answer for nothing.
		{"adapter/window/editor", load.Wire + "/gen/numen/v1", false},
		{"adapter/window/theme", load.Wire + "/gen/numen/v1", false},
		{"internal/testsupport", load.Module + "adapter/index", false},
	} {
		if got := isEdgeRefused(one.pkg, one.dep); got != one.isRefused {
			if one.isRefused {
				t.Errorf("%s reaches %s and is not rejected", one.pkg, one.dep)
			} else {
				t.Errorf("%s reaches %s and fails", one.pkg, one.dep)
			}
		}
	}

	// The composition root is the one package answering for what it names, and
	if _, named := getRefusedImports(load.Module + "container"); !named {
		t.Error("the composition root answers for everything it is built from")
	}
	if _, named := getRefusedImports(load.Module + "usecase/note"); named {
		t.Error("a scenario answers only for what it names")
	}
}

// Imports point inward. The test reads the build's resolved imports, so a
// package the core depends on through another package is checked as well.
//
// A test file is excluded: a test is outside the package it exercises and
// builds the adapters that replace the real ones.
func TestNothingTheCoreIsCompiledFromReachesOutward(t *testing.T) {
	for _, one := range load.ListShippedPlatforms() {
		t.Run(one.GOOS+"/"+one.GOARCH, func(t *testing.T) {
			inward(t, one.GOOS, one.GOARCH)
		})
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
