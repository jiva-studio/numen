package transports

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

// An inner layer defines the ports for what is outside and imports no transport,
// database driver or schema generator.
func TestInnerLayersDoNotImportTransports(t *testing.T) {
	var wrong []string
	var read int
	for _, root := range []string{"..", "../../../note/domain"} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			pkg := load.GetPackage("..", path)
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
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, one := range wrong {
		t.Error(one + ": the inner layers define ports and import no transport")
	}
	if read < 100 {
		t.Fatalf("%d files of the inner layers read: the walk is not reading them", read)
	}
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
