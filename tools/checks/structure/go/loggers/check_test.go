// Package loggers is the check `go/loggers`, run on the tree with the planted violations that prove it.
package loggers

import (
	"fmt"
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

// TestCorePackagesDoNotImportLoggers checks that nothing in the core logs, adapters and
// the composition root included.
// A log is a record for a reader who was not present; the user is in the
// window, and messages are shown there or on the process output stream.
func TestCorePackagesDoNotImportLoggers(t *testing.T) {
	wrong, err := findLoggerImports("..")
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one + ": nothing here logs — a person is told where they are")
	}
}

// TestLoggerWalk_FindsThePlantedFile verifies that the walk finds a file that imports a logger.
func TestLoggerWalk_FindsThePlantedFile(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"adapter/files/files.go": "package files\n\nimport \"log/slog\"\n\nvar _ = slog.Default\n",
		"usecase/note/note.go":   "package note\n",
	})
	wrong, err := findLoggerImports(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(wrong) != 1 || !strings.Contains(wrong[0], "adapter/files/files.go imports log/slog") {
		t.Errorf("the walk found %v, want the import of log/slog in adapter/files/files.go", wrong)
	}
}

// TestLoggerWalk_FailsOnAnEmptyTree verifies that a root without a Go file is an error, not a pass.
func TestLoggerWalk_FailsOnAnEmptyTree(t *testing.T) {
	if _, err := findLoggerImports(t.TempDir()); err == nil {
		t.Error("a walk over a tree without a Go file passed")
	}
}

// findLoggerImports lists the imports of a logger in the Go files under root. It fails when it reads no file.
func findLoggerImports(root string) ([]string, error) {
	var wrong []string
	var read int
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		read++
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				return err
			}
			if isLoggerPackage(to) {
				wrong = append(wrong, path+" imports "+to)
			}
		}
		return nil
	})
	if err == nil && read == 0 {
		err = fmt.Errorf("no Go file read under %s: the walk checks nothing", root)
	}
	return wrong, err
}

// find both of the standard library's and a third party's, and let through the
// packages whose names only begin the same way.
func TestLoggingRule_RejectsLoggingImports(t *testing.T) {
	var rejected []string
	for _, to := range []string{
		"log", "log/slog", "go.uber.org/zap", "github.com/rs/zerolog",
		"logic", "text/template", load.Module + "domain",
	} {
		if isLoggerPackage(to) {
			rejected = append(rejected, to)
		}
	}
	want := []string{"log", "log/slog", "go.uber.org/zap", "github.com/rs/zerolog"}
	if !slices.Equal(rejected, want) {
		t.Errorf("the rule rejected %v, want %v", rejected, want)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
