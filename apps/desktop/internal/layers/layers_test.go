// Package layers contains no production code. It holds the layering rules the
// applications are checked against: the core's own guard scans the core's tree,
// and an application is located one folder outside it.
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
const core = "github.com/jiva-studio/numen/libs/core/kernel/"

// apps is what every application is named under.
const apps = "github.com/jiva-studio/numen/apps/"

func isAssembled(pkg string) bool {
	return pkg == "container" || pkg == "domain" || strings.HasPrefix(pkg, "domain/") || pkg == "port" ||
		strings.HasPrefix(pkg, "port/") ||
		strings.HasPrefix(pkg, "adapter/") ||
		isContract(pkg)
}

func isContract(pkg string) bool {
	return strings.HasPrefix(pkg, "usecase/") && strings.HasSuffix(pkg, "contract")
}

// standing is a file of the applications, under the package it belongs to.
type standing struct {
	relPath string
	in      string
	file    *ast.File
}

func getRequiredFiles() []string {
	return []string{
		"desktop/cmd/numen/main.go",
		"desktop/internal/platform/platform.go",
		"mobile/bind/mobile.go",
	}
}

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
			relPath: filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator))),
			in:      getPackage(root, path),
			file:    file,
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
	for _, one := range getRequiredFiles() {
		if !slices.ContainsFunc(found, func(target standing) bool { return target.relPath == one }) {
			t.Fatalf("the walk did not read %s, so the rule stops before it", one)
		}
	}
	return found
}

// An application is served by the core and does none of its work, and no
// application imports another: what two of them share is a library.
func TestApplicationsDoNotImportCoreInternals(t *testing.T) {
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
				if getApplication(reaches) == getApplication(from) {
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
	if read < 40 {
		t.Fatalf("%d edges read: the walk is not reading the applications", read)
	}
}

func getLoggerPackages() map[string]bool {
	return map[string]bool{
		"log": true, "slog": true, "logrus": true, "zap": true,
		"zerolog": true, "logr": true, "glog": true, "klog": true,
	}
}

func isLoggerPackage(to string) bool { return getLoggerPackages()[to[strings.LastIndex(to, "/")+1:]] }

// Nothing here logs. An entry point names the process's own streams, and the
// one line it writes to standard error is for a terminal; everything the user
// has to act on is shown in the window. A logger would write to neither.
func TestApplicationsDoNotImportLoggers(t *testing.T) {
	var wrong []string
	for _, target := range readSources(t) {
		for _, one := range target.file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if isLoggerPackage(to) {
				wrong = append(wrong, target.relPath+" imports "+to)
			}
		}
	}
	for _, one := range wrong {
		t.Error(one + ": nothing here logs — a person is told where they are")
	}
}

// find both of the standard library's and a third party's, and let through the
// packages whose names only begin the same way.
func TestLoggingRule_RejectsLoggingImports(t *testing.T) {
	var violations []string
	for _, to := range []string{
		"log", "log/slog", "go.uber.org/zap", "github.com/rs/zerolog",
		"logic", "text/template", core + "domain",
	} {
		if isLoggerPackage(to) {
			violations = append(violations, to)
		}
	}
	want := []string{"log", "log/slog", "go.uber.org/zap", "github.com/rs/zerolog"}
	if !slices.Equal(violations, want) {
		t.Errorf("the rule rejects %v, want %v", violations, want)
	}
}

// A port is named after the need and its implementation after the technology,
// and the binding between them belongs to the composition root. A file that
// writes `var _ port.X = …` names the need it satisfies, which puts the binding
// in two places: when the port grows a method, this fails to compile where
// nothing yet requires that method.
func TestApplicationsDoNotClaimSatisfiedPorts(t *testing.T) {
	var wrong []string
	for _, target := range readSources(t) {
		for _, one := range getClaimedPorts(target.file) {
			wrong = append(wrong, target.relPath+" names port."+one)
		}
	}
	for _, one := range wrong {
		t.Error(one)
	}
}

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
func TestPortClaimRule_RejectsPortClaims(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "claim.go", `package p

var _ port.Indexer = (*Indexer)(nil)

var _, _ port.Trash = (*Bin)(nil), (*Bin)(nil)

var clockPort port.Clock

func mount() {
	var _ port.Recording = (*sound)(nil)
	var indexPort port.Indexer
	_, _ = clockPort, indexPort
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Indexer", "Trash", "Recording"}
	if got := getClaimedPorts(file); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
}

func getPackage(root, path string) string {
	target, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return path
	}
	return filepath.ToSlash(target)
}

func getApplication(pkg string) string {
	target, _, _ := strings.Cut(pkg, "/")
	return target
}
