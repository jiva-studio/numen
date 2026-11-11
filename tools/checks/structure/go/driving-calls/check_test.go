// Package drivingcalls is the check `go/driving-calls`, run on the tree with the planted violations that prove it.
package drivingcalls

import (
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
	fields := map[string][]string{}
	files := map[string]*ast.File{}
	structs := 0
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel := load.GetPackage("..", path) + "/" + entry.Name()
		if !isDrivingAdapterFile(rel) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		files[rel] = file
		structs += countStructTypes(file)
		dir := filepath.Dir(rel)
		fields[dir] = append(fields[dir], getPortFields(file)...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// A walk that read no driving adapter is a rule checked against nothing. A driving adapter holds no
	// port in a field, so no field proves the walk; the packages and struct types read do, and the fixture
	// test proves the rule fires.
	if len(files) < 30 {
		t.Fatalf("%d files of the driving adapters read: the walk is not reading them", len(files))
	}
	if packagesRead := len(fields); packagesRead < 15 {
		t.Fatalf("%d driving-adapter packages read, want at least 15: the walk is not reading them", packagesRead)
	}
	if structs < 60 {
		t.Fatalf("%d struct types of the driving adapters read, want at least 60: the walk is not reading them", structs)
	}

	for rel, file := range files {
		calls := getFieldCalls(file, fields[filepath.Dir(rel)])
		if len(calls) == 0 {
			continue
		}
		t.Errorf("%s calls %s: a driving adapter asks a use case, not a port", rel, calls[0])
	}
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
