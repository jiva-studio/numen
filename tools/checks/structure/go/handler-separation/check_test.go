package handlerseparation

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

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
