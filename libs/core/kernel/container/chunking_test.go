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
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/sourcefake"
)

// The sizes are part of what a chunk is kept under, and the one thing that
// assembles a cut carries them.
func TestTheSplitAssembledCarriesTheSizes(t *testing.T) {
	db, err := container.OpenIndex(container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(t.TempDir(), "index.db")},
	}), t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)

	target := container.NewConfig(container.Config{
		Paths: container.Paths{ServiceDir: ".numen"},
	})
	target.Resources.Chunks.Set(512)

	trimmed, err := container.IndexTexts(target, db, domain.Vault{ID: "v", Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if trimmed.Sizes != target.CutNotes().Sizes {
		t.Errorf("cut at %+v, and the settings say %+v", trimmed.Sizes, target.CutNotes().Sizes)
	}
}

func TestNoBoundSplitsAtTheDefaultBound(t *testing.T) {
	if got := (container.NewConfig(container.Config{})).CutNotes().Sizes.Limit; got != chunking.DefaultLimit {
		t.Errorf("got %+v", got)
	}
}

func TestSplitNotes_LimitFollowsTheChunkTokens(t *testing.T) {
	target := container.NewConfig(container.Config{})
	target.Resources.Chunks.Set(512)
	if got := target.CutNotes().Sizes.Limit; got != chunking.GetCharacterBound(512) {
		t.Errorf("cut at %d, under %d", got, chunking.GetCharacterBound(512))
	}
}

// A note is split at the sizes the settings define. The chunks a vault needs
// vectors for are its small ones, and each is under the input limit of the
// model that will embed it.
func TestANoteIsSplitAtTheSettingsSizes(t *testing.T) {
	db, err := container.OpenIndex(container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(t.TempDir(), "index.db")},
	}), t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)

	const maxTokens = 8
	target := container.NewConfig(container.Config{
		Paths: container.Paths{ServiceDir: ".numen"},
	})
	target.Resources.Chunks.Set(maxTokens)
	v := domain.Vault{ID: "v", Path: t.TempDir()}

	_, err = container.ReadWholeVault(target, t.Context(), db, sourcefake.Wrap(wide{384}), v)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Vaults().Register(t.Context(), v.ID); err != nil {
		t.Fatal(err)
	}

	body := strings.TrimSpace(strings.Repeat("chunks carry vectors ", 100))
	n := domain.Note{
		FileStat: domain.FileStat{Path: "notes/cut.md", Size: int64(len(body)), ModTime: time.Unix(0, 1)},
		Title:    "Cut",
		Body:     body,
	}
	if err := db.Notes().Save(t.Context(), v.ID, target.CutNotes().Execute([]domain.Note{n})); err != nil {
		t.Fatal(err)
	}

	unembedded, _, err := db.Sources().GetUnembeddedChunks(t.Context(), v.ID, wide{384}.Describe().Recipe(), "", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(unembedded) == 0 {
		t.Fatal("a note of a hundred lines needs no vector")
	}
	limit := chunking.GetCharacterBound(maxTokens)
	for _, p := range unembedded {
		if p.Length > limit {
			t.Errorf("a chunk of %d characters is embedded by a model that reads %d", p.Length, limit)
		}
	}
}

// The vector index is built for one width, and the width is the model's.
// Whichever entry point makes a vault searchable settles it.
func TestMakingAVaultSearchableFitsTheVectorIndex(t *testing.T) {
	db, err := container.OpenIndex(container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(t.TempDir(), "index.db")},
	}), t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)

	target := container.NewConfig(container.Config{
		Paths: container.Paths{ServiceDir: ".numen"},
	})
	v := domain.Vault{ID: "v", Path: t.TempDir()}

	if _, err := container.ReadWholeVault(target, t.Context(), db, sourcefake.Wrap(wide{384}), v); err != nil {
		t.Fatal(err)
	}
	// A width the index cannot be built for prevents anything from becoming
	// searchable, and it is detected before any vector is requested.
	_, err = container.ReadWholeVault(target, t.Context(), db, sourcefake.Wrap(wide{0}), v)
	if err == nil || !strings.Contains(err.Error(), "not a vector") {
		t.Fatalf("got %v", err)
	}
}

// wide is a model of the width given.
type wide struct{ dimensions int }

func (w wide) Describe() searchsource.Description {
	return searchsource.Description{Name: "wide", Width: w.dimensions, SourceRecipe: "mean"}
}

func (wide) Embed(context.Context, []string) ([][]float32, error) { return nil, nil }

func (wide) Close() error { return nil }

// Nothing outside this package builds a filetext.Index of its own, and no
// package but usecase/note, which owns the cutting, builds a chunking.Sizes or a
// chunking.Legibility. They decide what a chunk is kept under, and a second
// assembly is a second answer for one settings file.
func TestNothingElseAssemblesASplit(t *testing.T) {
	root := ".."
	isWithin := func(path, dir string) bool {
		return strings.HasPrefix(filepath.ToSlash(path), filepath.ToSlash(filepath.Join(root, dir))+"/")
	}
	var constructed, sized, untold []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if strings.HasSuffix(path, "_test.go") || isWithin(path, "container") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(n ast.Node) bool {
			// A note is split at the sizes the repository was configured with, so a
			// repository created without configuration splits at the package's defaults.
			if call, ok := n.(*ast.CallExpr); ok {
				if named, ok := call.Fun.(*ast.SelectorExpr); ok && named.Sel.Name == "Notes" {
					if target, ok := named.X.(*ast.Ident); ok && target.Name == "db" {
						untold = append(untold, path)
					}
				}
			}
			literal, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			named, ok := literal.Type.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := named.X.(*ast.Ident)
			if !ok {
				return true
			}
			switch {
			case pkg.Name == "filetext" && named.Sel.Name == "Index" &&
				!isWithin(path, "usecase/filetext"):
				constructed = append(constructed, path)
			case pkg.Name == "chunking" && (named.Sel.Name == "Sizes" || named.Sel.Name == "Legibility") &&
				!isWithin(path, "usecase/note"):
				sized = append(sized, path)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(constructed) != 0 {
		t.Errorf("a cut is assembled outside the composition root: %v", constructed)
	}
	if len(sized) != 0 {
		t.Errorf("a cut's sizes or thresholds are assembled outside usecase/note: %v", sized)
	}
	if len(untold) != 0 {
		t.Errorf("a note repository is taken without being told its sizes: %v", untold)
	}
}
