package search

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings/hardware"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// CoreFolder is the folder of a vault's store that holds the core's own files.
const CoreFolder = "core"

// ErrNoteUnreadable is the error of a note read again whose file could not be read.
var ErrNoteUnreadable = errors.New("the note could not be read")

// Config wires search, search policy, vectors and chunking of an installation.
type Config struct {
	ServiceDir         string
	ShouldRebuildIndex bool
	Chunks             *ChunkTokens
	ResolvedCompute    hardware.ResolvedCompute
	PluginHost         pluginport.PluginManager
	Clock              machineport.Clock
	VaultReaders       vaultport.VaultReaders
	TextReaders        textport.TextReaders
	ErrorHandler       func(error)
	ResolvePlugin      func(capability string) (pluginport.PluginManager, error)
}

// CutNotes is how a source's text and a note are cut into the chunks the index
// keeps: at this installation's sizes, keeping what reads as text.
func (c Config) CutNotes() chunking.CutNotes {
	cutter := query.NewCutNotes(c.Chunks.Get())
	if c.PluginHost != nil {
		cutter.Profiles = c.PluginHost.ResolveNoteProfile
	}
	return cutter
}

// SetNoteCut configures scan to cut notes at this installation's chunk sizes.
func (c Config) SetNoteCut(scan *vault.Scan, db *vault.Index) {
	scan.Cut = c.CutNotes()
	scan.ShouldRecut = c.Chunks.Get() > 0
	scan.GetOtherRecipePaths = func(ctx context.Context, v domain.VaultID, recipe string) ([]string, error) {
		recipes := []string{recipe, filetext.Index{Sizes: scan.Cut.Sizes}.Recipe()}
		return db.SourcesKnown().GetSourcesWithOtherRecipe(ctx, v, domain.KindNote, recipes, math.MaxInt32)
	}
}

// Scan is the walk that reads a vault's notes into the index, cut at this
// installation's sizes. Every application that reads a vault takes it from here.
func (c Config) Scan(db *vault.Index) vault.Scan {
	scan := vault.NewScan(
		c.VaultReaders,
		db.Vaults(),
		db.Notes(),
		db.Queries(),
		db.Maintenance(),
	)
	c.SetNoteCut(&scan, db)
	scan.Walks = db.Walks()
	scan.ShouldRebuildIndex = c.ShouldRebuildIndex
	return scan
}

// ReadWholeVault is what makes a vault answer, put together the one way: the
// notes read, the file texts cut, and the vectors made. Every entry point takes it
// from here, so a vault made searchable in a terminal and a vault made
// searchable in a window are the same vault.
func (c Config) ReadWholeVault(
	ctx context.Context, db *vault.Index, embedder embeddingport.Embedder, v domain.Vault, refresh vault.Refresh,
) (vaultreading.ReadWholeVault, error) {
	if err := c.FitIndex(db).Execute(ctx, embedder); err != nil {
		return vaultreading.ReadWholeVault{}, err
	}
	texts, err := c.IndexTexts(db.Sources(), db.SourcesKnown(), v)
	if err != nil {
		return vaultreading.ReadWholeVault{}, err
	}
	refresh.Force = true
	texts.ReadNote = NewNoteReader(refresh.Execute)
	embed, err := c.Embed(db, embedder, v)
	if err != nil {
		return vaultreading.ReadWholeVault{}, err
	}
	return vault.NewReadWholeVault(c.Scan(db), texts, embed), nil
}

// NewNoteReader adapts a vault refresh function into a single-note reader.
func NewNoteReader(
	refresh func(ctx context.Context, v domain.Vault, paths []string) (vault.RefreshResult, error),
) func(ctx context.Context, v domain.Vault, path string) error {
	return func(ctx context.Context, v domain.Vault, path string) error {
		res, err := refresh(ctx, v, []string{path})
		if err != nil {
			return err
		}
		if slices.Contains(res.Unreadable, path) {
			return fmt.Errorf("%s: %w", path, ErrNoteUnreadable)
		}
		return nil
	}
}

// FitIndex sets the vector index to the model of the embedding provider.
func (c Config) FitIndex(db *vault.Index) vectors.FitIndex {
	fit := vectors.NewFitIndex(db)
	fit.ShouldRebuildIndex = c.ShouldRebuildIndex
	fit.ErrorHandler = c.ErrorHandler
	return fit
}

// Embed gives a vault's chunks the vectors they need.
func (c Config) Embed(db *vault.Index, embedder embeddingport.Embedder, v domain.Vault) (vectors.Embed, error) {
	store, err := derivedfile.OpenDerived(v.Path, vaultconfig.Options{ServiceDir: c.ServiceDir}, CoreFolder)
	if err != nil {
		return vectors.Embed{}, err
	}
	embed := vectors.NewEmbed(c.VaultReaders, db.VectorQueries(), db.Vectors())
	embed.Claim = store.NewIndexClaim(CoreFolder + "/indexing")
	embed.Texts = c.TextReaders
	embed.Embedder = embedder
	return embed, nil
}

// IndexTexts cuts the text versions of a vault's files into chunks. Every entry
// point takes it from here, so what a chunk is kept under is one answer.
func (c Config) IndexTexts(sources sourceport.SourceRepository, queries sourceport.SourceQueries, v domain.Vault) (filetext.Index, error) {
	if _, err := c.VaultReaders.Open(v); err != nil {
		return filetext.Index{}, err
	}
	index := filetext.NewIndex(c.VaultReaders, sources, queries, c.TextReaders)
	index.ErrorHandler = c.ErrorHandler
	cutter := c.CutNotes()
	index.Sizes = cutter.Sizes
	index.Legibility = cutter.Legibility
	index.ShouldRebuildIndex = c.ShouldRebuildIndex
	return index, nil
}
