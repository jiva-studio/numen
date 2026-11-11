// Package transports is the check `go/transports`, run on the tree with the planted violations that prove it.
package transports

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

// An inner layer defines the ports for what is outside and imports no transport,
// database driver or schema generator.
func TestInnerLayersDoNotImportTransports(t *testing.T) {
	var wrong []string
	for _, root := range []string{"..", "../../../note/domain"} {
		found, err := findTransportImports(root, "..")
		if err != nil {
			t.Fatal(err)
		}
		wrong = append(wrong, found...)
	}
	for _, one := range wrong {
		t.Error(one + ": the inner layers define ports and import no transport")
	}
}

// TestTransportWalk_FindsThePlantedFiles verifies that the walk finds the transport import of a
// scenario and passes an adapter.
func TestTransportWalk_FindsThePlantedFiles(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"usecase/note/note.go":   "package note\n\nimport \"net/http\"\n\nvar _ = http.StatusOK\n",
		"adapter/files/files.go": "package files\n\nimport \"net/http\"\n\nvar _ = http.StatusOK\n",
	})
	wrong, err := findTransportImports(root, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(wrong) != 1 || !strings.Contains(wrong[0], "usecase/note/note.go imports net/http") {
		t.Errorf("the walk found %v, want the import of net/http in usecase/note/note.go only", wrong)
	}
}

// TestTransportWalk_FailsOnAnEmptyTree verifies that a root without a Go file is an error, not a pass.
func TestTransportWalk_FailsOnAnEmptyTree(t *testing.T) {
	if _, err := findTransportImports(t.TempDir(), "."); err == nil {
		t.Error("a walk over a tree without a Go file passed")
	}
}

// findTransportImports lists the transport imports of the files under dir outside the adapters, the
// shared packages and the machinery; a package is named by its path under base. It fails when it
// reads no file.
func findTransportImports(dir, base string) ([]string, error) {
	var wrong []string
	var read int
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		pkg := load.GetPackage(base, path)
		if load.IsAdapter(pkg) || load.IsShared(pkg) || load.IsMachinery(pkg) {
			return nil
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
			if isTransport(to) {
				wrong = append(wrong, path+" imports "+to)
			}
		}
		return nil
	})
	if err == nil && read == 0 {
		err = fmt.Errorf("no Go file read under %s: the walk checks nothing", dir)
	}
	return wrong, err
}

func TestTransportRule_RejectsTransportImports(t *testing.T) {
	var rejected []string
	for _, to := range []string{
		"connectrpc.com/connect",
		"database/sql",
		"database/sql/driver",
		"net/http",
		"net/http/httptest",
		"github.com/jiva-studio/numen/libs/core/protocol",
		"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1",
		"fmt",
		"context",
		load.Module + "port",
	} {
		if isTransport(to) {
			rejected = append(rejected, to)
		}
	}
	want := []string{
		"connectrpc.com/connect",
		"database/sql",
		"database/sql/driver",
		"net/http",
		"net/http/httptest",
		"github.com/jiva-studio/numen/libs/core/protocol",
		"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1",
	}
	if !slices.Equal(rejected, want) {
		t.Errorf("the rule rejected %v, want %v", rejected, want)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
