// Package ports is the check `go/ports`, run on the tree with the planted violations that prove it.
package ports

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The composition root hands out ports. An adapter the compiler holds is bound
// to one here and is never returned under its own name.
func TestTheRootPassesOutNoAdapterTheCompilerContains(t *testing.T) {
	target, err := load.ReadTree(".")
	if err != nil {
		t.Fatal(err)
	}
	read := 0
	for _, one := range target {
		name := one.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatal(err)
		}
		read++
		for _, why := range getRetainedAdapters(name, string(src)) {
			t.Errorf("%s: %s", name, why)
		}
	}

	if read == 0 {
		t.Fatalf("%d files of the composition root read: the walk is reading something else", read)
	}
	// The rule is about what the root answers with, so it is worth nothing where
	// the root answers nothing. Trash is a port handed out under its own name.
	if len(getRetainedAdapters("probe.go", `package container

import "`+load.Module+`port"

func (c Config) Trash() port.Trash { return nil }
`)) != 0 {
		t.Fatal("a port handed out under its own name fails: the rule is not the rule")
	}
}

// What the rule rejects, checked directly. Each case is a shape the walk meets
// in this package, so a failure here is the whole of the rule and not a sample.
func TestPortsRule_RejectsRootPackageAdapters(t *testing.T) {
	for _, one := range []struct {
		what      string
		src       string
		isRefused bool
	}{
		{
			what: "an adapter the compiler holds, answered under its own name",
			src: `package container
import "` + load.Module + `internal/adapter/filesystem"
func (c Config) Vaults() filesystem.VaultReaders { return filesystem.VaultReaders{} }`,
			isRefused: true,
		},
		{
			what: "the same, behind a pointer and inside a slice",
			src: `package container
import "` + load.Module + `internal/adapter/trash"
func (c Config) Cleaners() ([]*trash.Cleaner, error) { return nil, nil }`,
			isRefused: true,
		},
		{
			what: "a driving adapter the compiler holds: an application mounts it",
			src: `package container
import "` + load.Module + `adapter/window/theme"
func (c Config) Themes() *theme.Service { return nil }`,
			isRefused: false,
		},
		{
			what: "an adapter an application may name itself",
			src: `package container
import "` + load.Module + `adapter/settings"
func (c Config) Settings() (settings.Config, error) { return settings.Config{}, nil }`,
			isRefused: false,
		},
		{
			what: "the port the adapter satisfies",
			src: `package container
import "` + load.Module + `port"
func (c Config) Trash() port.Trash { return nil }`,
			isRefused: false,
		},
		{
			what: "an adapter the compiler holds, kept to this package",
			src: `package container
import "` + load.Module + `adapter/window/theme"
func (c Config) someTheme() (theme.Theme, error) {
	return theme.Theme{}, nil
}`,
			isRefused: false,
		},
		{
			what: "the same, on a receiver nothing outside can name",
			src: `package container
import "` + load.Module + `adapter/window/theme"
type appearances struct{}
func (a appearances) Read() (theme.Appearance, error) { return theme.Appearance{}, nil }`,
			isRefused: false,
		},
		{
			what: "an adapter named only in what the root is given",
			src: `package container
import "` + load.Module + `internal/adapter/filesystem"
func (c Config) Options(opts filesystem.Options) error { return nil }`,
			isRefused: false,
		},
	} {
		findings := getRetainedAdapters("probe.go", one.src)
		if one.isRefused && len(findings) == 0 {
			t.Errorf("%s is not rejected", one.what)
		}
		if !one.isRefused && len(findings) != 0 {
			t.Errorf("%s fails: %s", one.what, strings.Join(findings, "; "))
		}
	}
}

// TestCompositionRootDefinesNoAdapterStructs verifies that container/ is a pure Composition Root
// that wires adapters to ports and defines no inline adapter structs with methods of its own.
func TestCompositionRootDefinesNoAdapterStructs(t *testing.T) {
	entries, err := load.ReadTree(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		structTypes := map[string]bool{}
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if _, isStruct := ts.Type.(*ast.StructType); isStruct && ts.Name.Name != "Config" {
					structTypes[ts.Name.Name] = true
				}
			}
		}
		methodsByRecv := map[string][]string{}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
				continue
			}
			recv := fn.Recv.List[0].Type
			if star, ok := recv.(*ast.StarExpr); ok {
				recv = star.X
			}
			if id, ok := recv.(*ast.Ident); ok && structTypes[id.Name] {
				methodsByRecv[id.Name] = append(methodsByRecv[id.Name], fn.Name.Name)
			}
		}
		for st, methods := range methodsByRecv {
			t.Errorf("container/%s defines inline adapter struct %s with methods [%s]: move adapter and business logic out of Composition Root", name, st, strings.Join(methods, ", "))
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
