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
const wire = "github.com/jiva-studio/numen/libs/protocol"

// outward is what the core may not be compiled from: an adapter, the place
var outward = []string{
	module + "adapter", module + "internal/adapter", module + "container", wire,
}

// getRefusedImports is what a package of this module may not be compiled from,
// and whether it answers for what it names itself.
//
// The core answers for the whole of what it is built from. The composition root
// binds the adapters and is built from them, so it answers for what it names,
// and the schema is the one thing it may not name. An adapter and what only a
// test builds answer for nothing; so does a shared adapter library.
func getRefusedImports(path string) (refuses []string, named bool) {
	target := strings.TrimPrefix(path, module)
	switch {
	case target == "container":
		return []string{wire}, true
	case strings.HasPrefix(target, "adapter/"),
		strings.HasPrefix(target, "internal/adapter/"),
		strings.HasPrefix(target, "internal/test"),
		isShared(target):
		return nil, false
	}
	return outward, false
}

// isEdgeRefused says whether a package would be failed for the edge it has,
// which is what inward does with what getRefusedImports says.
func isEdgeRefused(pkg, dep string) bool {
	isRefused, _ := getRefusedImports(module + pkg)
	for _, blocked := range isRefused {
		if strings.HasPrefix(dep, blocked) {
			return true
		}
	}
	return false
}

// What getRefusedImports says is the whole of the rule, so it is asked directly.
// A walk that reads the tree and refuses nothing passes: these are the edges
func TestWhatTheDirectionRulesRefuse(t *testing.T) {
	for _, one := range []struct {
		pkg, dep  string
		isRefused bool
	}{
		// The core is compiled from none of the three, at any remove.
		{"usecase/note", module + "adapter/window/editor", true},
		{"usecase/note", module + "internal/adapter/theme", true},
		{"usecase/note", module + "container", true},
		{"usecase/note", wire + "/gen/numen/v1", true},
		{"text", module + "internal/adapter/filesystem", true},

		// The composition root is built from the adapters and answers for what
		// it names. The schema is the one thing it may not name.
		{"container", wire + "/gen/numen/v1", true},
		{"container", module + "adapter/index", false},
		{"container", module + "internal/adapter/theme", false},

		// does not.
		{"internal/adapterlib/wire", wire + "/gen/numen/v1", false},
		{"internal/adapterlib/onnxruntime", wire + "/gen/numen/v1", false},
		{"internal/query", wire + "/gen/numen/v1", true},

		// An adapter, and what only a test builds, answer for nothing.
		{"adapter/window/editor", wire + "/gen/numen/v1", false},
		{"internal/adapter/theme", wire + "/gen/numen/v1", false},
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

// shipped are the platforms this product is built for. A file kept for one of
// them is compiled nowhere else, so a listing read on the machine the test runs
// on is a rule checked against one platform's files. Every platform is read.
var shipped = []struct{ goos, goarch string }{
	{"linux", "amd64"},
	{"darwin", "arm64"},
	{"windows", "amd64"},
	{"android", "arm64"},
}

// Imports point inward. What the build resolved is what is read, so a package
// the core reaches through another answers for it too, and answering says how
//
// A test file is left out: a test stands outside the package it exercises and
// builds the adapters that stand in for the real ones.
func TestNothingTheCoreIsCompiledFromReachesOutward(t *testing.T) {
	for _, one := range shipped {
		t.Run(one.goos+"/"+one.goarch, func(t *testing.T) {
			inward(t, one.goos, one.goarch)
		})
	}
}

// inward reads the tree as one platform's build resolves it and holds every
// package of it to what answering says it may be compiled from.
func inward(t *testing.T, goos, goarch string) {
	t.Helper()
	listing := exec.CommandContext(t.Context(), "go", "list", "-json", "./...")
	listing.Dir = ".."
	listing.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch)
	items, err := listing.Output()
	if err != nil {
		var ran *exec.ExitError
		if errors.As(err, &ran) {
			t.Fatalf("go list: %v: %s", err, ran.Stderr)
		}
		t.Fatalf("go list: %v", err)
	}

	packages := 0
	decoder := json.NewDecoder(bytes.NewReader(items))
	for {
		var pkg struct {
			ImportPath string
			// Imports is what this package names, and Deps everything it is
			// compiled from, at whatever remove.
			Imports []string
			Deps    []string
		}
		err := decoder.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("go list: %v", err)
		}
		refuses, named := getRefusedImports(pkg.ImportPath)
		if refuses == nil {
			continue
		}
		packages++
		reaching := pkg.Deps
		if named {
			reaching = pkg.Imports
		}
		target := strings.TrimPrefix(pkg.ImportPath, module)
		for _, dep := range reaching {
			for _, blocked := range refuses {
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
