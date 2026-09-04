package container

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
)

// wire is the module the schema is generated into.
const wire = "github.com/jiva-studio/numen/modules/libs/protocol"

// outward is what the core may not be compiled from: an adapter, the place
// adapters are assembled, and the schema an adapter speaks.
var outward = []string{
	module + "adapter", module + "internal/adapter", module + "container", wire,
}

// owedWire are the edges to the schema the core still has. Each is a package
// naming the wire in the core's own language, and the list only shrinks.
var owedWire = map[string][]string{
	// What two adapters both put on the schema is built here.
	"internal/wire": {wire},
}

// owing says whether a package is allowed the edge it has.
func owing(from, dep string) bool {
	for _, held := range owedWire[from] {
		if strings.HasPrefix(dep, held) {
			return true
		}
	}
	return false
}

// answering is what a package of this module may not be compiled from, and
// whether it answers for what it names itself rather than for everything it is
// built from.
//
// The core answers for the whole of what it is built from. The composition root
// binds the adapters and is built from them, so it answers for what it names,
// and the schema is the one thing it may not name. An adapter and what only a
// test builds answer for nothing.
func answering(path string) (refuses []string, named bool) {
	held := strings.TrimPrefix(path, module)
	switch {
	case held == "container":
		return []string{wire}, true
	case strings.HasPrefix(held, "adapter/"),
		strings.HasPrefix(held, "internal/adapter/"),
		strings.HasPrefix(held, "internal/testsupport"):
		return nil, false
	}
	return outward, false
}

// Imports point inward. What the build resolved is what is read, so a package
// the core reaches through another answers for it too, and answering says how
// far each package is held to that.
//
// A test file is left out: a test stands outside the package it exercises and
// builds the adapters that stand in for the real ones.
func TestNothingTheCoreIsCompiledFromReachesOutward(t *testing.T) {
	listing := exec.CommandContext(t.Context(), "go", "list", "-json", "./...")
	listing.Dir = ".."
	listed, err := listing.Output()
	if err != nil {
		var ran *exec.ExitError
		if errors.As(err, &ran) {
			t.Fatalf("go list: %v: %s", err, ran.Stderr)
		}
		t.Fatalf("go list: %v", err)
	}

	packages := 0
	decoder := json.NewDecoder(bytes.NewReader(listed))
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
		refuses, named := answering(pkg.ImportPath)
		if refuses == nil {
			continue
		}
		packages++
		reaching := pkg.Deps
		if named {
			reaching = pkg.Imports
		}
		held := strings.TrimPrefix(pkg.ImportPath, module)
		for _, dep := range reaching {
			if owing(held, dep) {
				continue
			}
			for _, refused := range refuses {
				if strings.HasPrefix(dep, refused) {
					t.Errorf("%s reaches %s", held, strings.TrimPrefix(dep, module))
				}
			}
		}
	}

	// A listing of nothing is a rule checked against nothing.
	if packages == 0 {
		t.Fatal("go list found no package of the core")
	}
}
