package loggers

import (
	"go/parser"
	"go/token"
	"io/fs"
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
	var wrong []string
	var read int
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
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
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one + ": nothing here logs — a person is told where they are")
	}

	// A walk that read no file of the core is a rule checked against nothing,
	// and it passes.
	if read < 100 {
		t.Fatalf("%d files of the core read: the walk is not reading it", read)
	}
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
