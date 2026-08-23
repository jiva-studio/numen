package container_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/window"
)

// The sizes are part of what a chunk is kept under, and the one thing that
// assembles a cut carries them.
func TestTheCutAssembledCarriesTheSizes(t *testing.T) {
	cfg := embed.Defaults()
	cfg.Model.MaxTokens = 512
	held := container.Config{Embedding: cfg, ServiceDir: ".numen"}

	cut, err := held.Extract(nil, nil, domain.Vault{ID: "v", Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if cut.Sizes != held.Cutting() {
		t.Errorf("cut at %+v, and the settings say %+v", cut.Sizes, held.Cutting())
	}
}

func TestAVaultWithNoModelIsCutAtTheDefaultBound(t *testing.T) {
	cfg := embed.Defaults()
	cfg.Indexing.Use = ""
	if got := (container.Config{Embedding: cfg}).Cutting(); got != (window.Sizes{}) {
		t.Errorf("got %+v", got)
	}
}

func TestTheModelSaidIsWhatAWindowIsCutUnder(t *testing.T) {
	cfg := embed.Defaults()
	cfg.Model.MaxTokens = 512
	if got := (container.Config{Embedding: cfg}).Cutting().Limit; got != window.Under(512) {
		t.Errorf("cut at %d, under %d", got, window.Under(512))
	}
}

// Nothing outside this package builds a source.Extract of its own. The sizes it
// carries decide what a chunk is kept under, and a second assembly is a second
// answer for one settings file.
func TestNothingElseAssemblesACut(t *testing.T) {
	root := filepath.Join("..", "..", "internal")
	var built []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if strings.HasSuffix(path, "_test.go") || strings.Contains(path, "core/usecase/source") {
			return nil
		}
		if strings.HasPrefix(filepath.ToSlash(path), filepath.ToSlash(filepath.Join(root, "container"))) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if named, ok := lit.Type.(*ast.SelectorExpr); ok && named.Sel.Name == "Extract" {
				if pkg, ok := named.X.(*ast.Ident); ok && pkg.Name == "source" {
					built = append(built, path)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(built) != 0 {
		t.Errorf("a cut is assembled outside the composition root: %v", built)
	}
}
