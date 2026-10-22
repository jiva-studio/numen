// Package layers holds nothing. What is here is the rule the applications are
// read against: the core's own guard walks the core's tree, and an application
// is located one folder outside it.
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

// isAssembled is what an application takes from the core: the composition root,
// the adapters it serves something through, and the two languages both are
// named in. Everything else the core holds is the core's own work.
func isAssembled(pkg string) bool {
	return pkg == "container" || pkg == "domain" || pkg == "port" ||
		strings.HasPrefix(pkg, "adapter/")
}

// standing is a file of the applications, under the package it belongs to.
type standing struct {
	at   string
	in   string
	file *ast.File
}

// named are files every walk below has to have reached: the desktop's entry
// point, a file of its internal packages, and the phone. A count says how much was
// read and never what, and a walk that stopped at the desktop's border would
// clear any floor the desktop's own files fill.
var named = []string{
	"desktop/cmd/numen/main.go",
	"desktop/internal/platform/platform.go",
	"mobile/bind/mobile.go",
}

// readSources is every hand-written Go file of the applications, parsed.
//
// A test file is left out: a test stands outside the package it exercises and
func readSources(t *testing.T) []standing {
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
			in:   getPackage(root, path),
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
		if !slices.ContainsFunc(found, func(target standing) bool { return target.at == one }) {
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
	for _, target := range readSources(t) {
		from := target.in
		for _, one := range target.file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			switch {
			case strings.HasPrefix(to, core):
				read++
				reaches := strings.TrimPrefix(to, core)
				if isAssembled(reaches) {
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
	for _, target := range readSources(t) {
		for _, one := range target.file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if logs(to) {
				wrong = append(wrong, target.at+" imports "+to)
			}
		}
	}
	for _, one := range wrong {
		t.Error(one + ": nothing here logs — a person is told where they are")
	}
}

// find both of the standard library's and a third party's, and let through the
// packages whose names only begin the same way.
func TestWhatTheLoggingRuleRefuses(t *testing.T) {
	var violations []string
	for _, to := range []string{
		"log", "log/slog", "go.uber.org/zap", "github.com/rs/zerolog",
		"logic", "text/template", core + "domain",
	} {
		if logs(to) {
			violations = append(violations, to)
		}
	}
	want := []string{"log", "log/slog", "go.uber.org/zap", "github.com/rs/zerolog"}
	if !slices.Equal(violations, want) {
		t.Errorf("the rule rejects %v, want %v", violations, want)
	}
}

// A port is named after the need and what answers it after the technology, and
// the binding between them is the composition root's. A file that writes
// `var _ port.X = …` has named the need it answers, which puts the binding in
// two places: the day the port grows a method, this fails to compile where
// nothing yet asks it for that method.
func TestNoApplicationNamesThePortItSatisfies(t *testing.T) {
	var wrong []string
	for _, target := range readSources(t) {
		for _, one := range getClaimedPorts(target.file) {
			wrong = append(wrong, target.at+" names port."+one)
		}
	}
	for _, one := range wrong {
		t.Error(one)
	}
}

// getClaimedPorts are the ports a file declares itself to answer, by the blank
// name.
//
// A blank standing among other names is the same claim as one standing alone,
// and a declaration inside a function is the same claim as one beside the
// package's own. A variable that holds a port is not one: it is given what
// answers the port, and names what it was given.
func getClaimedPorts(file *ast.File) []string {
	var target []string
	ast.Inspect(file, func(node ast.Node) bool {
		decl, is := node.(*ast.GenDecl)
		if !is || decl.Tok != token.VAR {
			return true
		}
		for _, spec := range decl.Specs {
			named, is := spec.(*ast.ValueSpec)
			if !is || !slices.ContainsFunc(named.Names, func(pos *ast.Ident) bool {
				return pos.Name == "_"
			}) {
				continue
			}
			if offset, is := named.Type.(*ast.SelectorExpr); is {
				if from, is := offset.X.(*ast.Ident); is && from.Name == "port" {
					target = append(target, offset.Sel.Name)
				}
			}
		}
		return true
	})
	return target
}

// shapes of the claim are one claim, and the walk has to reach the one written
// inside a function; a variable holding a port is left alone.
func TestWhatThePortClaimRefuses(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "claim.go", `package p

var _ port.Agent = (*Agent)(nil)

var _, _ port.Trash = (*Bin)(nil), (*Bin)(nil)

var clockPort port.Clock

func mount() {
	var _ port.Recording = (*sound)(nil)
	var agentPort port.Agent
	_, _ = clockPort, agentPort
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Agent", "Trash", "Recording"}
	if got := getClaimedPorts(file); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
}

// getPackage is the package a file belongs to, as the rules name it.
func getPackage(root, path string) string {
	target, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return path
	}
	return filepath.ToSlash(target)
}

// application is the one a package belongs to.
func application(pkg string) string {
	target, _, _ := strings.Cut(pkg, "/")
	if target == "desktop-monolith" {
		return "desktop"
	}
	return target
}
