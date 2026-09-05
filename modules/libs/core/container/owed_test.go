package container

import (
	"bytes"
	"encoding/json"
	"errors"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// A list that only shrinks has to be made to shrink. An entry naming an edge
// the tree no longer has is a rule written down for nothing: it goes on
// admitting what it names, so the day somebody reaches for that edge again it
// is let through, and nobody decided that.
func TestNothingIsOwedThatIsNoLongerReached(t *testing.T) {
	for from, entries := range owed {
		for _, to := range entries {
			if !admitting(t, from, to) {
				t.Errorf("%s is owed %s and is refused nothing it reaches", from, to)
			}
		}
	}
}

// admitting says whether one entry of owed still lets an edge through: the
// entry is taken off the list, the package's own imports are read, and the rule
// is asked again. An entry that changes no answer is admitting nothing.
func admitting(t *testing.T, from, to string) bool {
	t.Helper()

	kept := owed[from]
	owed[from] = slices.DeleteFunc(slices.Clone(kept), func(one string) bool { return one == to })
	defer func() { owed[from] = kept }()

	for _, dep := range namedBy(t, from) {
		if dep != to && !strings.HasPrefix(dep, to+"/") {
			continue
		}
		if refused(from, dep) != "" {
			return true
		}
	}
	return false
}

// namedBy are the packages of this module one package's own files import. A
// test file is left out for the reason the rules leave it out.
func namedBy(t *testing.T, pkg string) []string {
	t.Helper()

	at := filepath.Join("..", filepath.FromSlash(pkg))
	held, err := os.ReadDir(at)
	if err != nil {
		t.Fatal(err)
	}
	var named []string
	for _, one := range held {
		if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") ||
			strings.HasSuffix(one.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(at, one.Name()), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range file.Imports {
			to, err := strconv.Unquote(imported.Path.Value)
			if err != nil || !strings.HasPrefix(to, module) {
				continue
			}
			named = append(named, strings.TrimPrefix(to, module))
		}
	}
	return named
}

// What the shrink rule refuses, read against entries written to be refused: one
// naming an edge the package does not have, and one naming an edge it has and
// is not refused for. The entries doing work have to come back admitting, or
// the walk is passing on an empty read.
func TestWhatTheOwedShrinkRuleRefuses(t *testing.T) {
	for _, one := range []struct {
		from, to string
		admits   bool
	}{
		{"adapter/cli", "container", true},
		{"container", "task", true},
		{"adapter/cli", "internal/adapter/trash", false},
		{"adapter/settings", "port", false},
	} {
		if got := admitting(t, one.from, one.to); got != one.admits {
			if one.admits {
				t.Errorf("%s is owed %s and the rule finds nothing it admits", one.from, one.to)
			} else {
				t.Errorf("%s is owed %s and the rule calls it a debt", one.from, one.to)
			}
		}
	}
}

// The edges to the generated schema are the same list under another rule, and
// the same holds of them: an entry no package answers to is taken off.
func TestNothingIsOwedTheSchemaThatIsNoLongerNamed(t *testing.T) {
	for _, one := range unpaid(owedWire, builtFrom(t)) {
		t.Error(one + ": nothing it is compiled from is the schema")
	}
}

// unpaid are the entries of an owed list naming an edge the package it stands
// under does not have.
func unpaid(owing, built map[string][]string) []string {
	var idle []string
	for from, entries := range owing {
		for _, to := range entries {
			if !slices.ContainsFunc(built[from], func(dep string) bool {
				return strings.HasPrefix(dep, to)
			}) {
				idle = append(idle, from+" is owed "+to)
			}
		}
	}
	slices.Sort(idle)
	return idle
}

// builtFrom is everything each package of the core is compiled from, at
// whatever remove, as this machine's build resolves it.
func builtFrom(t *testing.T) map[string][]string {
	t.Helper()

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

	built := map[string][]string{}
	decoder := json.NewDecoder(bytes.NewReader(listed))
	for {
		var pkg struct {
			ImportPath string
			Deps       []string
		}
		err := decoder.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("go list: %v", err)
		}
		built[strings.TrimPrefix(pkg.ImportPath, module)] = pkg.Deps
	}

	// A listing of nothing is a rule checked against nothing.
	if len(built) == 0 {
		t.Fatal("go list found no package of the core")
	}
	return built
}

// What the rule refuses, read against a listing written to be refused. The
// entry the listing answers is left alone and the one it does not is named.
func TestWhatTheOwedSchemaRuleRefuses(t *testing.T) {
	built := map[string][]string{
		"internal/wire": {module + "port", wire + "/gen/numen/v1"},
		"container":     {module + "port"},
	}
	got := unpaid(map[string][]string{
		"internal/wire": {wire},
		"container":     {wire},
	}, built)
	if want := []string{"container is owed " + wire}; !slices.Equal(got, want) {
		t.Errorf("the rule refuses %v, want %v", got, want)
	}
}
