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
const wire = "github.com/jiva-studio/numen/modules/libs/protocol"

// outward is what the core may not be compiled from: an adapter, the place
// adapters are assembled, and the schema an adapter speaks.
var outward = []string{
	module + "adapter", module + "internal/adapter", module + "container", wire,
}

// wireBaseline are the edges to the schema the core still has. Each is a
// package naming the wire in the core's own language, and the list only
// shrinks.
var wireBaseline = map[string][]string{
	// What two adapters both put on the schema is built here.
	"internal/wire": {wire},
}

// baselined says whether a package is allowed the edge it has.
func baselined(from, dep string) bool {
	for _, held := range wireBaseline[from] {
		if strings.HasPrefix(dep, held) {
			return true
		}
	}
	return false
}

// answering is what a package of this module may not be compiled from, and
// whether it answers for what it names itself.
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

// reaching says whether a package would be failed for the edge it has, which is
// what inward does with what answering and baselined say.
func reaching(pkg, dep string) bool {
	refuses, _ := answering(module + pkg)
	if baselined(pkg, dep) {
		return false
	}
	for _, refused := range refuses {
		if strings.HasPrefix(dep, refused) {
			return true
		}
	}
	return false
}

// What answering and baselined say between them is the whole of the rule, so
// they are asked directly. A walk that reads the tree and refuses nothing
// passes, and an entry admitting one edge that quietly admitted every edge
// would pass with it: these are the edges that have to come back refused.
func TestWhatTheDirectionRulesRefuse(t *testing.T) {
	for _, one := range []struct {
		pkg, dep string
		refuses  bool
	}{
		// The core is compiled from none of the three, at any remove.
		{"usecase/note", module + "adapter/window/editor", true},
		{"usecase/note", module + "internal/adapter/theme", true},
		{"usecase/note", module + "container", true},
		{"usecase/note", wire + "/gen/numen/v1", true},
		{"usecase/note", module + "domain", false},
		{"domain", module + "adapter/index", true},
		{"text", module + "internal/adapter/pdf", true},

		// The composition root is built from the adapters and answers for what
		// it names. The schema is the one thing it may not name.
		{"container", wire + "/gen/numen/v1", true},
		{"container", module + "adapter/index", false},
		{"container", module + "internal/adapter/theme", false},

		// What two adapters both put on the schema is built in one place, and
		// the edge that admits it admits nothing else.
		{"internal/wire", wire + "/gen/numen/v1", false},
		{"internal/wire", module + "adapter/window/editor", true},

		// An adapter, and what only a test builds, answer for nothing.
		{"adapter/window/editor", wire + "/gen/numen/v1", false},
		{"internal/adapter/theme", wire + "/gen/numen/v1", false},
		{"internal/testsupport", module + "adapter/index", false},
	} {
		if got := reaching(one.pkg, one.dep); got != one.refuses {
			if one.refuses {
				t.Errorf("%s reaches %s and is not refused", one.pkg, one.dep)
			} else {
				t.Errorf("%s reaches %s and is refused", one.pkg, one.dep)
			}
		}
	}

	// The composition root is the one package answering for what it names, and
	// everything else is held to the whole of what it is compiled from.
	if _, named := answering(module + "container"); !named {
		t.Error("the composition root answers for everything it is built from")
	}
	if _, named := answering(module + "usecase/note"); named {
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
// far each package is held to that.
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
			if baselined(held, dep) {
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
