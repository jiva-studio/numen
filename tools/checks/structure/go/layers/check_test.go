package layers

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The adapters depend on the core and the core depends on none of them; an adapter is
// given what it needs and takes no other adapter; assembly happens in one place.
//
// Test files are excluded: a test is external to the package it exercises and
// builds the adapters that stand in for the real ones.
func TestLayerDependenciesPointInward(t *testing.T) {
	var wrong []string
	var read int
	root := ".."
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		from := load.GetPackage(root, path)
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil || !strings.HasPrefix(to, load.Module) {
				continue
			}
			read++
			if why := getRefusal(from, strings.TrimPrefix(to, load.Module)); why != "" {
				wrong = append(wrong, from+" reaches "+strings.TrimPrefix(to, load.Module)+": "+why)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}

	// A walk that reads no edge of this module checks nothing and passes. The count
	// is a lower bound well below the core's edge count; it confirms the walk found
	// the tree, not how large the tree is.
	if read < 100 {
		t.Fatalf("%d edges of this module read: the walk is not reading the core", read)
	}
}

// TestNoPackageIsImportedBelowItsLayer verifies that packages do not use layer names as import aliases.
func TestNoPackageIsImportedBelowItsLayer(t *testing.T) {
	var wrong []string
	var read, aliased int
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		read++
		for _, one := range file.Imports {
			if one.Name == nil {
				continue
			}
			aliased++
			if !getLayers()[one.Name.Name] {
				continue
			}
			target, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				return err
			}
			wrong = append(wrong, path+" names "+target+" "+one.Name.Name)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}

	// A walk that read no file, or one that met no alias, is a rule checked
	// against nothing, and it passes: this rule has only aliases to read.
	if read < 100 {
		t.Fatalf("%d files of the core read: the walk is not reading it", read)
	}
	if aliased < 10 {
		t.Fatalf("%d imports carry a name of their own: the walk is reading no alias", aliased)
	}
}

// TestLayerAliasRule_RejectsLayerNamedAliases verifies rejection of layer names as import aliases.
func TestLayerAliasRule_RejectsLayerNamedAliases(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "alias.go", `package p

import (
	"maps"
	usecase "`+load.Module+`usecase/note"
	adapter "`+load.Module+`adapter/index"
	vaults "`+load.Module+`usecase/vault"
	"`+load.Module+`domain"
	pathpkg "path"
)
`, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	var rejected []string
	for _, one := range file.Imports {
		if one.Name != nil && getLayers()[one.Name.Name] {
			rejected = append(rejected, one.Name.Name)
		}
	}
	if want := []string{"usecase", "adapter"}; !slices.Equal(rejected, want) {
		t.Errorf("the rule rejected %v, want %v", rejected, want)
	}
}

// holds took the name of one it does not and was given that name's baseline
// entry, and a package holding what is true of a note reached one that holds
// work.
func TestLayerRule_RefusesForbiddenEdges(t *testing.T) {
	for _, one := range []struct {
		from, to  string
		isRefused bool
	}{
		// The window is given what it needs and assembles nothing, and one
		// settings file is the union of sections each adapter owns.
		{"adapter/window/editor", "container", true},
		{"adapter/settings", "internal/adapter/filesystem", true},

		// The folder holding one adapter to a window is not an adapter, so the
		// two windows are two adapters and neither is given the other.
		{"adapter/window/editor", "adapter/window/shell", true},
		{"adapter/window/shell", "adapter/window/editor", true},
		{"adapter/window/editor/pages", "adapter/window/editor", false},

		// A driven adapter runs no scenario and takes no other adapter. Which of the
		// two an adapter is does not depend on its folder: the compiler-restricted
		// folder here also serves the schema.
		{"adapter/index", "usecase/note", true},
		{"internal/adapter/filesystem", "usecase/note", true},
		{"adapter/window/theme", "usecase/note", false},
		{"adapter/window/theme", "internal/adapter/filesystem", true},
		{"internal/adapter/themefiles", "usecase/note", true},
		{"internal/adapter/themefiles", "adapter/window/theme", true},
		{"adapter/window/theme", "internal/adapter/themefiles", true},
		{"usecase/theme", "internal/adapter/themefiles", true},
		{"adapter/index/chunk", "adapter/index", false},

		// The core imports no adapter and no package that assembles one.
		{"usecase/note", "internal/adapter/trash", true},
		{"usecase/note", "container", true},
		{"usecase/note", "container/vault", true},
		{"usecase/note", "containerized", false},
		{"usecase/note", "port", false},
		{"internal/adapterlib/wire", "adapter/window/editor", true},
		{"internal/adapterlib/wire", "container", true},
		{"internal/adapterlib/wire", "internal/adapter/trash", true},
		{"internal/adapterlib/wire", "usecase/note", false},
		{"internal/adapterlib/onnxruntime", "usecase/note", true},
		{"internal/adapterlib/onnxruntime", "internal/adapter/filesystem", true},
		{"internal/adapterlib/onnxruntime", "port", false},

		// A domain package of the core imports no other layer of it, and every layer
		// above it may import the domain.
		{"domain/plugin", "port", true},
		{"domain/plugin", "port/pluginport", true},
		{"domain/plugin", "usecase/note", true},
		{"domain/plugin", "adapter/index", true},
		{"domain/plugin", "internal/adapter/trash", true},
		{"domain/plugin", "internal/text", true},
		{"domain/plugin", "container", true},
		{"domain/plugin", "domain/settings", false},
		{"domain/settings/sub", "domain/settings", false},
		{"container", "domain/plugin", false},
		{"usecase/note", "domain/plugin", false},
		{"port/pluginport", "domain/plugin", false},
		{"adapter/window/editor", "domain/plugin", false},
		{"internal/adapter/plugin", "domain/plugin", false},

		// Fixtures build the real adapters.
		{"internal/testsupport", "adapter/index", false},
		{"internal/testonly", "container", false},

		// The composition root does none of the core's work.
		{"container", "internal/adapterlib/wire", true},
		{"container", "internal/text", true},
		{"container", "adapter/index", false},
		{"container", "usecase/run", false},
	} {
		why := getRefusal(one.from, one.to)
		if one.isRefused && why == "" {
			t.Errorf("%s reaches %s and is not rejected", one.from, one.to)
		}
		if !one.isRefused && why != "" {
			t.Errorf("%s reaches %s and is rejected: %s", one.from, one.to, why)
		}
	}
}

// Reading the tree is what the rules are checked against, so the tree has to be
// where this expects it.
func TestLayerDirectoriesExist(t *testing.T) {
	for _, item := range []string{"domain", "port", "usecase", "adapter", "internal/adapter", "internal/adapterlib"} {
		if _, err := os.Stat(load.GetPackageDir(item)); err != nil {
			t.Fatalf("%s: %v", item, err)
		}
	}
	// getPackageDir reads "domain" as the notes domain; the domain packages of the core are read here.
	if _, err := os.Stat(filepath.Join("..", "domain")); err != nil {
		t.Fatal(err)
	}
}
