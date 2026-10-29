package notes_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const notesModule = "github.com/jiva-studio/numen/libs/notes/"

var allowedInternalEdges = map[string][]string{
	"domain":   nil,
	"ulid":     nil,
	"markdown": {"domain", "ulid"},
	"edit":     {"domain", "ulid", "markdown"},
}

var forbiddenInfra = map[string]bool{
	"os":            true,
	"os/exec":       true,
	"net":           true,
	"net/http":      true,
	"database/sql":  true,
	"encoding/json": true,
	"syscall":       true,
}

func TestNotesInternalHexagonalLayers(t *testing.T) {
	read := 0
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		dir := filepath.ToSlash(filepath.Dir(path))
		pkg, _, _ := strings.Cut(dir, "/")
		if _, isListed := allowedInternalEdges[pkg]; !isListed {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		read++
		for _, imp := range file.Imports {
			to, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}
			if forbiddenInfra[to] {
				t.Errorf("%s imports %s: libs/notes is a pure domain/text library and performs no I/O or wire serialization", path, to)
			}
			if after, ok := strings.CutPrefix(to, notesModule); ok {
				target, _, _ := strings.Cut(after, "/")
				if target != pkg && !slices.Contains(allowedInternalEdges[pkg], target) {
					t.Errorf("%s (%s) imports %s (%s): violates internal layer hierarchy edit -> markdown -> {domain, ulid}", path, pkg, to, target)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if read < 10 {
		t.Fatalf("only %d files read in libs/notes", read)
	}
}
