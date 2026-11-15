// Package schemadriving is the check `go/schema-driving`, run on the tree with the planted violations that prove it.
package schemadriving

import (
	"go/parser"
	"go/token"
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// TestEveryAdapterServingTheSchemaIsDriving ensures adapters serving or mounting
// generated protocol handlers are among the driving adapters load.GetDriving reads from the tree.
func TestEveryAdapterServingTheSchemaIsDriving(t *testing.T) {
	wrong, serving, err := findDrivenSchemaServers("..")
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}
	if serving == 0 {
		t.Fatal("no adapter names the schema: the walk is not reading the adapters")
	}
}

// The walk rejects a driven adapter that mounts a handler, and passes a driving adapter that mounts one, a driven
// adapter that only calls a client, a test file and a package outside the adapters.
func TestSchemaDrivingRule_RejectsADrivenAdapterThatServes(t *testing.T) {
	serve := "package p\n\nimport \"" + load.Schema + "\"\n\nfunc mount(h any) { numenv1connect.NewNoteServiceHandler(h) }\n"
	call := "package p\n\nimport \"" + load.Schema + "\"\n\nfunc call() { numenv1connect.NewNoteServiceClient(nil, \"\") }\n"
	root := load.WriteTreeFixture(t, map[string]string{
		"internal/adapter/notes/notes.go":      serve,
		"adapter/cli/cli.go":                   serve,
		"internal/adapter/remote/remote.go":    call,
		"internal/adapter/other/other_test.go": serve,
		"usecase/note/note.go":                 serve,
	})
	wrong, serving, err := findDrivenSchemaServers(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"internal/adapter/notes serves the schema and is named driven"}
	if !slices.Equal(wrong, want) {
		t.Errorf("the walk rejects %v, want %v", wrong, want)
	}
	if serving != 2 {
		t.Errorf("%d serving files read, want 2", serving)
	}
}

// The rule counts a package that names a handler type or constructor of the connect package, and
// lets through one that only builds messages or calls a client.
func TestHandlerRule_SeparatesServingFromCalling(t *testing.T) {
	for _, tc := range []struct {
		what, body string
		isServing  bool
	}{
		{"a handler constructor", `numenv1connect.NewNoteServiceHandler(h)`, true},
		{"a handler type", `var _ numenv1connect.NoteServiceHandler`, true},
		{"a client constructor", `numenv1connect.NewNoteServiceClient(nil, "")`, false},
		{"a service name", `_ = numenv1connect.NoteServiceName`, false},
	} {
		source := "package p\n\nimport \"" + load.Schema + "\"\n\nfunc f(h any) {\n\t" + tc.body + "\n}\n"
		file, err := parser.ParseFile(token.NewFileSet(), "p.go", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		if got := load.NamesHandler(file); got != tc.isServing {
			t.Errorf("%s: serving is %v, want %v", tc.what, got, tc.isServing)
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
