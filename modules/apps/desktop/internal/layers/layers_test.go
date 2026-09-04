// Package layers holds nothing. What is here is the rule the applications are
// read against, which had been read for the core alone: the core's own guard
// walks the core's tree, so an application standing one folder outside it was
// held to nothing at all.
package layers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// core is what every package of the core is named under.
const core = "github.com/jiva-studio/numen/modules/libs/core/"

// apps is what every application is named under.
const apps = "github.com/jiva-studio/numen/modules/apps/"

// assembled is what an application takes from the core: the composition root,
// the adapters it serves something through, and the two languages both are
// named in. Everything else the core holds is the core's own work.
func assembled(pkg string) bool {
	return pkg == "container" || pkg == "domain" || pkg == "port" ||
		strings.HasPrefix(pkg, "adapter/")
}

// owed are the edges the applications still have. Each is a binary doing a
// piece of the core's work where it should be handed the whole, and the list
// only shrinks: an edge missing from the tree is not an error here, so the
// composition root taking one back needs no line changed.
var owed = map[string][]string{
	// The window builds the agent's tool surface itself, so it reads a note,
	// a card and the rules over both in place.
	"desktop/cmd/numen": {
		"check", "flashcards/format", "markdown", "usecase/note",
	},
	// The review window builds its own surface, its own queue and its own
	// vault opening.
	"desktop/cmd/numen-flashcards": {
		"flashcards/format", "task", "usecase/flashcards", "usecase/vault",
	},
	// The proofreading adapter speaks the port's own language.
	"desktop/internal/adapter/claudecode": {"proofread"},
	// The binding registers the vault it seeded.
	"mobile/bind": {"usecase/vault"},
}

// An application is served by the core and does none of its work, and no
// application reaches another: what two of them share is a library.
//
// A test file is left out: a test stands outside the package it exercises and
// builds what stands in for the real thing.
func TestNoApplicationDoesTheCoresWork(t *testing.T) {
	var wrong []string
	var read int
	root := filepath.Join("..", "..", "..")
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			if entry != nil && entry.IsDir() && entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		from := within(root, path)
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				return err
			}
			switch {
			case strings.HasPrefix(to, core):
				read++
				held := strings.TrimPrefix(to, core)
				if assembled(held) || allowed(from, held) {
					continue
				}
				wrong = append(wrong, from+" reaches "+held+
					": an application is served by the core and does none of its work")
			case strings.HasPrefix(to, apps):
				read++
				held := strings.TrimPrefix(to, apps)
				if application(held) == application(from) {
					continue
				}
				wrong = append(wrong, from+" reaches "+held+
					": what two applications share is a library")
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

	// A walk that read no edge is a rule checked against nothing, and it
	// passes. The floor is well under what the applications hold.
	if read < 50 {
		t.Fatalf("%d edges read: the walk is not reading the applications", read)
	}
}

// A port is named after the need and what answers it after the technology, and
// the binding between them is the composition root's. A file that writes
// `var _ port.X = …` has named the need it answers, which puts the binding in
// two places: the day the port grows a method, this fails to compile where
// nothing yet asks it for that method.
func TestNoApplicationNamesThePortItSatisfies(t *testing.T) {
	var wrong []string
	err := filepath.WalkDir(filepath.Join("..", "..", ".."), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			if entry != nil && entry.IsDir() && entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, one := range claimed(file) {
			wrong = append(wrong, path+" names port."+one)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}
}

// claimed are the ports a file declares itself to answer, by the blank name.
func claimed(file *ast.File) []string {
	var held []string
	for _, one := range file.Decls {
		decl, is := one.(*ast.GenDecl)
		if !is || decl.Tok != token.VAR {
			continue
		}
		for _, spec := range decl.Specs {
			named, is := spec.(*ast.ValueSpec)
			if !is || len(named.Names) != 1 || named.Names[0].Name != "_" {
				continue
			}
			if at, is := named.Type.(*ast.SelectorExpr); is {
				if from, is := at.X.(*ast.Ident); is && from.Name == "port" {
					held = append(held, at.Sel.Name)
				}
			}
		}
	}
	return held
}

// within is the package a file belongs to, as the rules name it.
func within(root, path string) string {
	held, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return path
	}
	return filepath.ToSlash(held)
}

// application is the one a package belongs to.
func application(pkg string) string {
	held, _, _ := strings.Cut(pkg, "/")
	return held
}

// allowed says whether a package is owed the edge it has.
func allowed(from, to string) bool {
	for _, held := range owed[from] {
		if to == held || strings.HasPrefix(to, held+"/") {
			return true
		}
	}
	return false
}
