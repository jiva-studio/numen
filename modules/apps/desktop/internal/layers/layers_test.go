// Package layers holds nothing. What is here is the rule the applications are
// read against: the core's own guard walks the core's tree, and an application
// stands one folder outside it.
package layers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
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
	// The review window builds its own surface and its own queue.
	"desktop/cmd/numen-flashcards": {
		"flashcards/format", "task", "usecase/flashcards",
	},
	// The proofreading adapter speaks the port's own language.
	"desktop/internal/adapter/claudecode": {"proofread"},
	// The binding registers the vault it seeded.
	"mobile/bind": {"usecase/vault"},
}

// standing is a file of the applications, under the package it belongs to.
type standing struct {
	at   string
	in   string
	file *ast.File
}

// named are files every walk below has to have reached: the desktop's entry
// point, the deepest file it holds, and the phone. A count says how much was
// read and never what, and a walk that stopped at the desktop's border would
// clear any floor the desktop's own files fill.
var named = []string{
	"desktop/cmd/numen/main.go",
	"desktop/internal/adapter/claudecode/claudecode.go",
	"mobile/bind/mobile.go",
}

// walked is every hand-written Go file of the applications, parsed.
//
// A test file is left out: a test stands outside the package it exercises and
// builds what stands in for the real thing.
func walked(t *testing.T) []standing {
	t.Helper()

	var found []standing
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
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		found = append(found, standing{
			at:   filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator))),
			in:   within(root, path),
			file: file,
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// A walk that read nothing is a rule checked against nothing, and it
	// passes. The floor is under what the applications hold, and the files
	// above are named besides.
	if len(found) < 20 {
		t.Fatalf("%d files read: the walk is not reading the applications", len(found))
	}
	for _, one := range named {
		if !slices.ContainsFunc(found, func(held standing) bool { return held.at == one }) {
			t.Fatalf("the walk did not read %s, so the rule stops before it", one)
		}
	}
	return found
}

// An application is served by the core and does none of its work, and no
// application reaches another: what two of them share is a library.
func TestNoApplicationDoesTheCoresWork(t *testing.T) {
	var wrong []string
	var read int
	for _, held := range walked(t) {
		from := held.in
		for _, one := range held.file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			switch {
			case strings.HasPrefix(to, core):
				read++
				reaches := strings.TrimPrefix(to, core)
				if assembled(reaches) || allowed(from, reaches) {
					continue
				}
				wrong = append(wrong, from+" reaches "+reaches+
					": an application is served by the core and does none of its work")
			case strings.HasPrefix(to, apps):
				read++
				reaches := strings.TrimPrefix(to, apps)
				if application(reaches) == application(from) {
					continue
				}
				wrong = append(wrong, from+" reaches "+reaches+
					": what two applications share is a library")
			}
		}
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

// logging are the words a package that keeps a record is named by, read off the
// last element of an import path. The standard library holds two of them, and
// the rest are the libraries reached for when one of those will not do.
var logging = map[string]bool{
	"log": true, "slog": true, "logrus": true, "zap": true,
	"zerolog": true, "logr": true, "glog": true, "klog": true,
}

// logs answers whether an import is a logger.
func logs(to string) bool { return logging[to[strings.LastIndex(to, "/")+1:]] }

// Nothing here logs. An entry point is where the process's own streams are
// named, and the one line it writes to standard error is for a terminal
// somebody is standing at; everything a person has to act on is said in the
// window they are looking at. A logger would write to neither.
func TestNoApplicationLogs(t *testing.T) {
	var wrong []string
	for _, held := range walked(t) {
		for _, one := range held.file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if logs(to) {
				wrong = append(wrong, held.at+" imports "+to)
			}
		}
	}
	for _, one := range wrong {
		t.Error(one + ": nothing here logs — a person is told where they are")
	}
}

// What the rule refuses, read against imports written to be refused. It has to
// find both of the standard library's and a third party's, and let through the
// packages whose names only begin the same way.
func TestWhatTheLoggingRuleRefuses(t *testing.T) {
	var refused []string
	for _, to := range []string{
		"log", "log/slog", "go.uber.org/zap", "github.com/rs/zerolog",
		"logic", "text/template", core + "domain",
	} {
		if logs(to) {
			refused = append(refused, to)
		}
	}
	want := []string{"log", "log/slog", "go.uber.org/zap", "github.com/rs/zerolog"}
	if !slices.Equal(refused, want) {
		t.Errorf("the rule refuses %v, want %v", refused, want)
	}
}

// A port is named after the need and what answers it after the technology, and
// the binding between them is the composition root's. A file that writes
// `var _ port.X = …` has named the need it answers, which puts the binding in
// two places: the day the port grows a method, this fails to compile where
// nothing yet asks it for that method.
func TestNoApplicationNamesThePortItSatisfies(t *testing.T) {
	var wrong []string
	for _, held := range walked(t) {
		for _, one := range claimed(held.file) {
			wrong = append(wrong, held.at+" names port."+one)
		}
	}
	for _, one := range wrong {
		t.Error(one)
	}
}

// claimed are the ports a file declares itself to answer, by the blank name.
//
// A blank standing among other names is the same claim as one standing alone,
// and a declaration inside a function is the same claim as one beside the
// package's own. A variable that holds a port is not one: it is given what
// answers the port, and names what it was given.
func claimed(file *ast.File) []string {
	var held []string
	ast.Inspect(file, func(node ast.Node) bool {
		decl, is := node.(*ast.GenDecl)
		if !is || decl.Tok != token.VAR {
			return true
		}
		for _, spec := range decl.Specs {
			named, is := spec.(*ast.ValueSpec)
			if !is || !slices.ContainsFunc(named.Names, func(at *ast.Ident) bool {
				return at.Name == "_"
			}) {
				continue
			}
			if at, is := named.Type.(*ast.SelectorExpr); is {
				if from, is := at.X.(*ast.Ident); is && from.Name == "port" {
					held = append(held, at.Sel.Name)
				}
			}
		}
		return true
	})
	return held
}

// What the rule refuses, read against a file written to be refused. The three
// shapes of the claim are one claim, and the walk has to reach the one written
// inside a function; a variable holding a port is left alone.
func TestWhatThePortClaimRefuses(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "claim.go", `package p

var _ port.Agent = (*Agent)(nil)

var _, _ port.Trash = (*Bin)(nil), (*Bin)(nil)

var held port.Clock

func mount() {
	var _ port.Recording = (*sound)(nil)
	var kept port.Agent
	_, _ = held, kept
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Agent", "Trash", "Recording"}
	if got := claimed(file); !slices.Equal(got, want) {
		t.Errorf("the rule refuses %v, want %v", got, want)
	}
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
