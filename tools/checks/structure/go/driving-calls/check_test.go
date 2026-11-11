// Package drivingcalls is the check `go/driving-calls`, run on the tree with the planted violations that prove it.
package drivingcalls

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// A driving adapter calls a use case. The composition root supplies any port
// held in a field, and a call through that field would perform the use case's
// work in the adapter.
func TestNoDrivingAdapterCallsAPortOutsideItsWiring(t *testing.T) {
	wrong, err := findPortFieldCalls("..")
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}
}

// TestDrivingWalk_FindsThePlantedCall verifies that the walk finds the call through a port field in
// a driving adapter and passes the same call in a package that is no driving adapter.
func TestDrivingWalk_FindsThePlantedCall(t *testing.T) {
	source := "package p\n\ntype API struct{ Queries noteport.NoteRefs }\n\nfunc read(a *API) { a.Queries.GetTypes() }\n"
	root := load.WriteTreeFixture(t, map[string]string{
		"adapter/window/window.go": source,
		"usecase/note/note.go":     source,
	})
	wrong, err := findPortFieldCalls(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(wrong) != 1 || !strings.Contains(wrong[0], "adapter/window/window.go calls Queries.GetTypes") {
		t.Errorf("the walk found %v, want the call in adapter/window/window.go only", wrong)
	}
}

// TestDrivingWalk_FailsOnATreeWithoutDrivingAdapters verifies that a root without a driving adapter is an error, not a pass.
func TestDrivingWalk_FailsOnATreeWithoutDrivingAdapters(t *testing.T) {
	if _, err := findPortFieldCalls(t.TempDir()); err == nil {
		t.Error("a walk over a tree without a driving adapter passed")
	}
}

func findPortFieldCalls(root string) ([]string, error) {
	// Fails when it reads no driving adapter.
	fields := map[string][]string{}
	files := map[string]*ast.File{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel := load.GetPackage(root, path) + "/" + entry.Name()
		if !isDrivingAdapterFile(rel) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		files[rel] = file
		dir := filepath.Dir(rel)
		fields[dir] = append(fields[dir], getPortFields(file)...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no driving-adapter file read under %s: the walk checks nothing", root)
	}
	var wrong []string
	for rel, file := range files {
		calls := getFieldCalls(file, fields[filepath.Dir(rel)])
		if len(calls) == 0 {
			continue
		}
		wrong = append(wrong, fmt.Sprintf("%s calls %s: a driving adapter asks a use case, not a port", rel, calls[0]))
	}
	slices.Sort(wrong)
	return wrong, nil
}

// call through a field typed port.X, and lets through a call through a field of
// another type and a call on a value that is no field.
func TestDrivingCallRule_RejectsPortFieldCalls(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "call.go", `package p

type Notes struct {
	Queries noteport.NoteRefs
	Links   *noteport.LinkQueries
	Find    usecase.Find
}

func read(a *API, queries noteport.NoteRefs) {
	a.Notes.Queries.GetTypes()
	a.Notes.Links.Resolve()
	a.Notes.Find.Execute()
	queries.GetTypes()
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	fields := getPortFields(file)
	if want := []string{"Queries", "Links"}; !slices.Equal(fields, want) {
		t.Fatalf("the port fields are %v, want %v", fields, want)
	}
	want := []string{"Queries.GetTypes", "Links.Resolve"}
	if got := getFieldCalls(file, fields); !slices.Equal(got, want) {
		t.Errorf("the rule refuses %v, want %v", got, want)
	}
}

// Every driving adapter is on the lint rule that keeps it off the ports, with zero ! file exemptions.
func TestEveryDrivingAdapterIsOnTheLintRule(t *testing.T) {
	patterns := depguardFiles(t)
	if len(patterns) == 0 {
		t.Fatal("the driving-adapter rule lists no file: the config is not being read")
	}
	for _, p := range patterns {
		if strings.HasPrefix(p, "!") && p != "!$test" {
			t.Errorf("driving-adapter depguard rule contains file exemption %s: all driving adapters must go through use cases", p)
		}
	}
	for _, one := range getUncovered(patterns) {
		t.Errorf("%s is a driving adapter and the depguard driving-adapter rule does not list it", one)
	}
}

func TestLintCoverageRule_ReportsUncoveredAdapters(t *testing.T) {
	got := getUncovered([]string{"**/adapter/window/**", "**/adapter/plugin/**", "**/internal/adapterlib/wire/**", "!**/adapter/cli/deps.go"})
	if want := []string{"adapter/cli"}; !slices.Equal(got, want) {
		t.Errorf("the rule refuses %v, want %v", got, want)
	}
	if got := getUncovered([]string{"**/adapter/window/**", "**/adapter/plugin/**", "**/internal/adapterlib/wire/**", "**/adapter/cli/**"}); len(got) != 0 {
		t.Errorf("the rule refuses %v, want none", got)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
