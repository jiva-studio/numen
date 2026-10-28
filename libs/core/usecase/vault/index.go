package vault

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/indexport"
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/problemport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

// IndexBackend represents lifecycle and vector-maintenance operations of an index store.
type IndexBackend interface {
	Close() error
	FitVectors(ctx context.Context, dims int, recipe string) error
	DeleteOtherRecipeVectors(ctx context.Context, recipe string) (int64, error)
	Compact(ctx context.Context) error
}

// IndexPorts bundles the ports and backend of an open index database.
type IndexPorts struct {
	Backend       IndexBackend
	Path          string
	Vaults        vaultport.VaultRepository
	Notes         noteport.NoteRepository
	Queries       noteport.NoteQueries
	Passages      sourceport.PassageQueries
	Progress      indexport.IndexProgress
	Sources       sourceport.SourceRepository
	SourcesKnown  sourceport.SourceQueries
	Vectors       embeddingport.VectorRepository
	VectorQueries embeddingport.VectorQueries
	Maintenance   indexport.IndexMaintenance
	Links         noteport.LinkQueries
	Problems      problemport.ProblemQueries
	OpenTabs      windowport.OpenTabsStore
	TextVersions  textport.VersionRepositories
}

// Index is the cache, seen as the ports the core asks for.
type Index struct {
	ports  IndexPorts
	walks  Walks
	ledger *Ledger
}

// NewIndex constructs an Index facade over the provided IndexPorts.
func NewIndex(ports IndexPorts) *Index {
	return &Index{ports: ports, ledger: NewLedger()}
}

func (i *Index) Close() error { return i.ports.Backend.Close() }

// Walks is the turns the vaults walked into this index take.
func (i *Index) Walks() *Walks { return &i.walks }

// Ledger returns the shared refresh ledger of this index.
func (i *Index) Ledger() *Ledger { return i.ledger }

// FitVectors makes the vector index hold vectors of the width given.
func (i *Index) FitVectors(ctx context.Context, dims int, recipe string) error {
	return i.ports.Backend.FitVectors(ctx, dims, recipe)
}

// DeleteVectorsOfOtherRecipes takes out the vectors kept under any recipe but the one in use.
func (i *Index) DeleteVectorsOfOtherRecipes(ctx context.Context, recipe string) (int64, error) {
	return i.ports.Backend.DeleteOtherRecipeVectors(ctx, recipe)
}

// Compact returns the space the index no longer holds.
func (i *Index) Compact(ctx context.Context) error { return i.ports.Backend.Compact(ctx) }

func (i *Index) Vaults() vaultport.VaultRepository           { return i.ports.Vaults }
func (i *Index) Notes() noteport.NoteRepository              { return i.ports.Notes }
func (i *Index) Queries() noteport.NoteQueries               { return i.ports.Queries }
func (i *Index) Passages() sourceport.PassageQueries         { return i.ports.Passages }
func (i *Index) Progress() indexport.IndexProgress           { return i.ports.Progress }
func (i *Index) Sources() sourceport.SourceRepository        { return i.ports.Sources }
func (i *Index) SourcesKnown() sourceport.SourceQueries      { return i.ports.SourcesKnown }
func (i *Index) Vectors() embeddingport.VectorRepository     { return i.ports.Vectors }
func (i *Index) VectorQueries() embeddingport.VectorQueries  { return i.ports.VectorQueries }
func (i *Index) Maintenance() indexport.IndexMaintenance     { return i.ports.Maintenance }
func (i *Index) Path() string                                { return i.ports.Path }
func (i *Index) Links() noteport.LinkQueries                 { return i.ports.Links }
func (i *Index) Problems() problemport.ProblemQueries        { return i.ports.Problems }
func (i *Index) OpenTabs() windowport.OpenTabsStore          { return i.ports.OpenTabs }
func (i *Index) TextVersions() textport.VersionRepositories  { return i.ports.TextVersions }
