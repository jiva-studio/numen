package container

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// wire is the module the schema is generated into.
const wire = "github.com/jiva-studio/numen/libs/core/protocol"

func listOutward() []string {
	return []string{
		module + "adapter", module + "internal/adapter", module + "container", wire,
	}
}

func getRefusedImports(path string) (forbidden []string, named bool) {
	target := strings.TrimPrefix(path, module)
	switch {
	case isUnder(target, "container"):
		return []string{wire}, true
	case strings.HasPrefix(target, "adapter/"),
		strings.HasPrefix(target, "internal/adapter/"),
		strings.HasPrefix(target, "internal/test"),
		isShared(target):
		return nil, false
	}
	return listOutward(), false
}

func isEdgeRefused(pkg, dep string) bool {
	isRefused, _ := getRefusedImports(module + pkg)
	for _, blocked := range isRefused {
		if strings.HasPrefix(dep, blocked) {
			return true
		}
	}
	return false
}

// getRefusedImports defines the complete rule, so the test calls it directly.
// A walk that reads the tree but rejects nothing would pass, so these cases pin
// the forbidden edges.
func TestWhatTheDirectionRulesRefuse(t *testing.T) {
	for _, one := range []struct {
		pkg, dep  string
		isRefused bool
	}{
		// The core is compiled from none of the three, at any remove.
		{"usecase/note", module + "adapter/window/editor", true},
		{"usecase/note", module + "adapter/window/theme", true},
		{"usecase/note", module + "container", true},
		{"usecase/note", wire + "/gen/numen/v1", true},
		{"text", module + "internal/adapter/filesystem", true},

		// The composition root is built from the adapters and answers for what
		// it names. The schema is the one thing it may not name.
		{"container", wire + "/gen/numen/v1", true},
		{"container", module + "adapter/index", false},
		{"container", module + "adapter/window/theme", false},

		// does not.
		{"internal/adapterlib/wire", wire + "/gen/numen/v1", false},
		{"internal/adapterlib/onnxruntime", wire + "/gen/numen/v1", false},
		{"internal/query", wire + "/gen/numen/v1", true},

		// An adapter, and what only a test builds, answer for nothing.
		{"adapter/window/editor", wire + "/gen/numen/v1", false},
		{"adapter/window/theme", wire + "/gen/numen/v1", false},
		{"internal/testsupport", module + "adapter/index", false},
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
	if _, named := getRefusedImports(module + "container"); !named {
		t.Error("the composition root answers for everything it is built from")
	}
	if _, named := getRefusedImports(module + "usecase/note"); named {
		t.Error("a scenario answers only for what it names")
	}
}

func listShippedPlatforms() []struct{ goos, goarch string } {
	return []struct{ goos, goarch string }{
		{"linux", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
		{"android", "arm64"},
	}
}

// Imports point inward. The test reads the build's resolved imports, so a
// package the core depends on through another package is checked as well.
//
// A test file is excluded: a test is outside the package it exercises and
// builds the adapters that replace the real ones.
func TestNothingTheCoreIsCompiledFromReachesOutward(t *testing.T) {
	for _, one := range listShippedPlatforms() {
		t.Run(one.goos+"/"+one.goarch, func(t *testing.T) {
			inward(t, one.goos, one.goarch)
		})
	}
}

func inward(t *testing.T, goos, goarch string) {
	t.Helper()
	listing := exec.CommandContext(t.Context(), "go", "list", "-json", "./...")
	listing.Dir = ".."
	listing.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch)
	items, err := listing.Output()
	if err != nil {
		if ran, ok := errors.AsType[*exec.ExitError](err); ok {
			t.Fatalf("go list: %v: %s", err, ran.Stderr)
		}
		t.Fatalf("go list: %v", err)
	}

	packages := 0
	decoder := json.NewDecoder(bytes.NewReader(items))
	for {
		var pkg struct {
			ImportPath string   `json:"ImportPath"`
			Imports    []string `json:"Imports"`
			Deps       []string `json:"Deps"`
		}
		err := decoder.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("go list: %v", err)
		}
		forbidden, named := getRefusedImports(pkg.ImportPath)
		if forbidden == nil {
			continue
		}
		packages++
		reaching := pkg.Deps
		if named {
			reaching = pkg.Imports
		}
		target := strings.TrimPrefix(pkg.ImportPath, module)
		for _, dep := range reaching {
			for _, blocked := range forbidden {
				if strings.HasPrefix(dep, blocked) {
					t.Errorf("%s reaches %s", target, strings.TrimPrefix(dep, module))
				}
			}
		}
	}

	// A listing of nothing is a rule checked against nothing.
	if packages == 0 {
		t.Fatal("go list found no package of the core")
	}
}
