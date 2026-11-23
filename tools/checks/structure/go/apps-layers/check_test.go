// Package appslayers is the check `go/apps-layers`, run on the tree with the planted violations that prove it. An
// application imports only the assembled packages of the core, no other application, no logger, names no port it
// satisfies and links no plugin.
package appslayers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// core is what every package of the core is named under.
const core = "github.com/jiva-studio/numen/libs/core/kernel/"

// apps is what every application is named under.
const apps = "github.com/jiva-studio/numen/apps/"

func getAppsDir() string { return filepath.Join(load.RepoRootDir, "apps") }

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

func readSources(t *testing.T, root string) []standing {
	t.Helper()

	var found []standing
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
	return found
}

func readApplications(t *testing.T) []standing {
	t.Helper()
	found := readSources(t, getAppsDir())
	for _, one := range getRequiredFiles() {
		if !slices.ContainsFunc(found, func(target standing) bool { return target.relPath == one }) {
			t.Fatalf("the walk did not read %s, so the rule stops before it", one)
		}
	}
	return found
}

func findCoreReaches(t *testing.T, sources []standing) ([]string, int) {
	t.Helper()
	var wrong []string
	var read int
	for _, target := range sources {
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
	return wrong, read
}

// An application is served by the core and does none of its work, and no
// application imports another: what two of them share is a library.
func TestApplicationsDoNotImportCoreInternals(t *testing.T) {
	wrong, read := findCoreReaches(t, readApplications(t))
	for _, one := range wrong {
		t.Error(one)
	}
	if read == 0 {
		t.Fatal("no import of the core or of an application read: the walk is not reading the applications")
	}
}

// The rule rejects an import of a use case of the core and of another application, and passes an assembled
// package of the core, a package of the same application and a test file.
func TestCoreReachRule_RejectsCoreInternalsAndOtherApplications(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"desktop/app/app.go": "package app\n\nimport (\n" +
			"\t_ \"" + core + "usecase/note\"\n" +
			"\t_ \"" + core + "container\"\n" +
			"\t_ \"" + core + "usecase/notecontract\"\n" +
			"\t_ \"" + apps + "mobile/bind\"\n" +
			"\t_ \"" + apps + "desktop/internal/platform\"\n)\n",
		"desktop/app/app_test.go": "package app\n\nimport _ \"" + core + "usecase/search\"\n",
	})
	wrong, read := findCoreReaches(t, readSources(t, root))
	want := []string{
		"desktop/app reaches usecase/note: an application is served by the core and does none of its work",
		"desktop/app reaches mobile/bind: what two applications share is a library",
	}
	if !slices.Equal(wrong, want) {
		t.Errorf("the rule rejects %v, want %v", wrong, want)
	}
	if read != 5 {
		t.Errorf("%d imports read, want 5", read)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.RunTests(m))
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
	for _, target := range readApplications(t) {
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
	for _, target := range readApplications(t) {
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
