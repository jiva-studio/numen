package container_test

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/cutting"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
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
	if got := (container.Config{Embedding: cfg}).Cutting(); got != (cutting.Sizes{}) {
		t.Errorf("got %+v", got)
	}
}

func TestTheModelSaidIsWhatAChunkIsCutUnder(t *testing.T) {
	cfg := embed.Defaults()
	cfg.Model.MaxTokens = 512
	if got := (container.Config{Embedding: cfg}).Cutting().Limit; got != cutting.Under(512) {
		t.Errorf("cut at %d, under %d", got, cutting.Under(512))
	}
}

// A note is cut at the sizes the settings say. The chunks a vault owes vectors
// for are its small ones, and each is under the input limit of the model that
// will read it.
func TestANoteIsCutAtTheSettingsSizes(t *testing.T) {
	db, err := container.Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	cfg := embed.Defaults()
	cfg.Model.MaxTokens = 8
	held := container.Config{Embedding: cfg, ServiceDir: ".numen"}
	v := domain.Vault{ID: "v", Path: t.TempDir()}

	searchable, err := held.Searchable(t.Context(), db, wide{384}, v)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Vaults().Save(t.Context(), v); err != nil {
		t.Fatal(err)
	}

	body := strings.TrimSpace(strings.Repeat("chunks carry vectors ", 100))
	n := domain.Note{
		Ref:   domain.FileRef{Path: "notes/cut.md", Size: int64(len(body)), MTime: 1},
		Title: "Cut",
		Body:  body,
	}
	if err := searchable.Notes.Notes.Save(t.Context(), v.ID, []domain.Note{n}); err != nil {
		t.Fatal(err)
	}

	owing, err := db.VectorsOwing().Unembedded(t.Context(), v.ID, wide{384}.Model(), 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) == 0 {
		t.Fatal("a note of a hundred lines owes no vector")
	}
	limit := cutting.Under(cfg.Model.MaxTokens)
	for _, p := range owing {
		if p.Length > limit {
			t.Errorf("a chunk of %d characters is embedded by a model that reads %d", p.Length, limit)
		}
	}
}

// The vector index is built for one width, and the width is the model's.
// Whichever entry point makes a vault searchable settles it.
func TestMakingAVaultSearchableFitsTheVectorIndex(t *testing.T) {
	db, err := container.Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	held := container.Config{Embedding: embed.Defaults(), ServiceDir: ".numen"}
	v := domain.Vault{ID: "v", Path: t.TempDir()}

	if _, err := held.Searchable(t.Context(), db, wide{384}, v); err != nil {
		t.Fatal(err)
	}
	// A width the index cannot be built for is the reason nothing is made
	// searchable, and it is found before a vector is asked of anything.
	_, err = held.Searchable(t.Context(), db, wide{0}, v)
	if err == nil || !strings.Contains(err.Error(), "not a vector") {
		t.Fatalf("got %v", err)
	}
}

// wide is a model of the width given.
type wide struct{ dimensions int }

func (w wide) Model() port.EmbeddingModel {
	return port.EmbeddingModel{Name: "wide", Dimensions: w.dimensions, MaxTokens: 256, Pooling: "mean"}
}

func (wide) Embed(context.Context, []string) ([][]float32, error) { return nil, nil }

func (wide) Close() error { return nil }

// Nothing outside this package builds a source.Extract or a cutting.Sizes of its
// own. The sizes decide what a chunk is kept under, and a second assembly is a
// second answer for one settings file.
func TestNothingElseAssemblesACut(t *testing.T) {
	root := filepath.Join("..", "..", "internal")
	within := func(path, dir string) bool {
		return strings.HasPrefix(filepath.ToSlash(path), filepath.ToSlash(filepath.Join(root, dir))+"/")
	}
	var built, sized, untold []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if strings.HasSuffix(path, "_test.go") || within(path, "container") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(n ast.Node) bool {
			// A note is cut by whatever the repository was told when it was
			// built, so a repository taken without being told cuts at the
			// package's own defaults.
			if call, ok := n.(*ast.CallExpr); ok {
				if named, ok := call.Fun.(*ast.SelectorExpr); ok && named.Sel.Name == "Notes" {
					if held, ok := named.X.(*ast.Ident); ok && held.Name == "db" {
						untold = append(untold, path)
					}
				}
			}
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			named, ok := lit.Type.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := named.X.(*ast.Ident)
			if !ok {
				return true
			}
			switch {
			case pkg.Name == "source" && named.Sel.Name == "Extract" && !within(path, "core/usecase/source"):
				built = append(built, path)
			case pkg.Name == "cutting" && named.Sel.Name == "Sizes":
				sized = append(sized, path)
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
	if len(sized) != 0 {
		t.Errorf("sizes are assembled outside the composition root: %v", sized)
	}
	if len(untold) != 0 {
		t.Errorf("a note repository is taken without being told its sizes: %v", untold)
	}
}
