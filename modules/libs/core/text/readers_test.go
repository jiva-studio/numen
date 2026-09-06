package text_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/text"
)

// Every reader this package declares is one Readers answers with.
//
// A recipe names the reader that produced a source's text, and what asks which
// sources owe their text asks it with the recipe of every reader. A reader
// declared and left out of the list is one no recipe of its ever matches: every
// source it reads is selected as owing its text on every run, read again,
// stamped with the same unmatched recipe, and selected again — a library
// re-extracted for ever, with nothing said anywhere.
func TestEveryReaderDeclaredIsOneReadersAnswersWith(t *testing.T) {
	answered := text.Readers()
	for name, value := range declared(t) {
		if !slices.Contains(answered, value) {
			t.Errorf("%s is %q, and Readers() answers with %q", name, value, answered)
		}
	}
}

// declared is every `Reader…` constant this package declares, by name. The
// source is what is read: a constant nothing names is a constant reflection
// cannot reach.
func declared(t *testing.T) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]string{}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			spec, ok := node.(*ast.ValueSpec)
			if !ok {
				return true
			}
			for i, ident := range spec.Names {
				if !strings.HasPrefix(ident.Name, "Reader") || i >= len(spec.Values) {
					continue
				}
				if literal, ok := spec.Values[i].(*ast.BasicLit); ok && literal.Kind == token.STRING {
					found[ident.Name] = strings.Trim(literal.Value, `"`)
				}
			}
			return true
		})
	}
	if len(found) == 0 {
		t.Fatal("no reader was found declared in this package")
	}
	return found
}
