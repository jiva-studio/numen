package container

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// An adapter the compiler holds is one an application cannot name. Where the
// composition root answers with one, the application holds a value it has no
// word for: it cannot declare a variable of that type, cannot write a stand-in
// for it in a test, and cannot see in its own source what it was given.
//
// A driven adapter therefore comes back as the port it satisfies. A driving one
// is what an application mounts and calls in through, and has no port to come
// back as; those are named in driving and are not this rule's business.
const compilerHolds = module + "internal/adapter/"

// getHeldAdapters is every place src answers with an adapter the compiler
// holds, said in full. A method on an unexported receiver is left out: it is
// not the root's answer to anybody outside.
func getHeldAdapters(name, src string) []string {
	f, err := parser.ParseFile(token.NewFileSet(), name, src, 0)
	if err != nil {
		return []string{name + " does not parse: " + err.Error()}
	}

	under := map[string]string{}
	for _, one := range f.Imports {
		path, err := strconv.Unquote(one.Path.Value)
		if err != nil {
			continue
		}
		at := path[strings.LastIndex(path, "/")+1:]
		if one.Name != nil {
			at = one.Name.Name
		}
		under[at] = path
	}

	var said []string
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !fn.Name.IsExported() || fn.Type.Results == nil || !reachable(fn) {
			continue
		}
		for _, result := range fn.Type.Results.List {
			ast.Inspect(result.Type, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				at, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				path := under[at.Name]
				if !strings.HasPrefix(path, compilerHolds) || driving[strings.TrimPrefix(path, module)] {
					return true
				}
				said = append(said, fn.Name.Name+" answers with "+at.Name+"."+sel.Sel.Name+
					", which is "+strings.TrimPrefix(path, module)+"'s own type and not a port")
				return true
			})
		}
	}
	return said
}

// reachable is whether a declaration is one something outside the package can
// reach: a plain function, or a method on a type it can name.
func reachable(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return true
	}
	on := fn.Recv.List[0].Type
	if star, ok := on.(*ast.StarExpr); ok {
		on = star.X
	}
	name, ok := on.(*ast.Ident)
	return ok && name.IsExported()
}

// The composition root hands out ports. An adapter the compiler holds is bound
// to one here and never comes back out under its own name.
func TestTheRootHandsOutNoAdapterTheCompilerHolds(t *testing.T) {
	held, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	read := 0
	for _, one := range held {
		name := one.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatal(err)
		}
		read++
		for _, why := range getHeldAdapters(name, string(src)) {
			t.Errorf("%s: %s", name, why)
		}
	}

	// A walk that read nothing is a rule checked against nothing, and it passes.
	// The floor is well under what this package holds.
	if read < 10 {
		t.Fatalf("%d files of the composition root read: the walk is reading something else", read)
	}
	// The rule is about what the root answers with, so it is worth nothing where
	// the root answers nothing. Trash is a port handed out under its own name.
	if len(getHeldAdapters("probe.go", `package container

import "`+module+`port"

func (c Config) Trash() port.Trash { return nil }
`)) != 0 {
		t.Fatal("a port handed out under its own name is refused: the rule is not the rule")
	}
}

// What the rule refuses, asked directly. Each case is a shape the walk meets in
// this package, so a refusal here is the whole of the rule and not a sample.
func TestWhatThePortRuleRefusesTheRoot(t *testing.T) {
	for _, one := range []struct {
		what    string
		src     string
		refuses bool
	}{
		{
			what: "an adapter the compiler holds, answered under its own name",
			src: `package container
import "` + module + `internal/adapter/filesystem"
func (c Config) Vaults() filesystem.VaultReaders { return filesystem.VaultReaders{} }`,
			refuses: true,
		},
		{
			what: "the same, behind a pointer and inside a slice",
			src: `package container
import "` + module + `internal/adapter/embed"
func (c Config) Embedders() ([]*embed.Model, error) { return nil, nil }`,
			refuses: true,
		},
		{
			what: "a driving adapter the compiler holds: an application mounts it",
			src: `package container
import "` + module + `internal/adapter/theme"
func (c Config) Themes() *theme.Service { return nil }`,
			refuses: false,
		},
		{
			what: "an adapter an application may name itself",
			src: `package container
import "` + module + `adapter/settings"
func (c Config) Settings() (settings.Config, error) { return settings.Config{}, nil }`,
			refuses: false,
		},
		{
			what: "the port the adapter satisfies",
			src: `package container
import "` + module + `port"
func (c Config) Trash() port.Trash { return nil }`,
			refuses: false,
		},
		{
			what: "an adapter the compiler holds, kept to this package",
			src: `package container
import "` + module + `internal/adapter/proofreading"
func (c Config) profile(name string) (proofreading.Profile, error) {
	return proofreading.Profile{}, nil
}`,
			refuses: false,
		},
		{
			what: "the same, on a receiver nothing outside can name",
			src: `package container
import "` + module + `internal/adapter/theme"
type appearances struct{}
func (a appearances) Read() (theme.Appearance, error) { return theme.Appearance{}, nil }`,
			refuses: false,
		},
		{
			what: "an adapter named only in what the root is given",
			src: `package container
import "` + module + `internal/adapter/filesystem"
func (c Config) Options(held filesystem.Options) error { return nil }`,
			refuses: false,
		},
	} {
		said := getHeldAdapters("probe.go", one.src)
		if one.refuses && len(said) == 0 {
			t.Errorf("%s is not refused", one.what)
		}
		if !one.refuses && len(said) != 0 {
			t.Errorf("%s is refused: %s", one.what, strings.Join(said, "; "))
		}
	}
}
