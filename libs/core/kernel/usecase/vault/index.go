package vault

import (
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/indexport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/problemport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchpolicyport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
)

// IndexStores are the repositories of the vaults and their notes, the tabs open in a window and the
// maintenance of the index.
type IndexStores struct {
	Vaults      vaultport.VaultRepository
	Notes       noteport.NoteRepository
	OpenTabs    windowport.OpenTabsStore
	Maintenance indexport.IndexMaintenance
}

// IndexNoteQueries are the questions the index answers about the notes of a vault.
type IndexNoteQueries struct {
	FileStats   noteport.FileStats
	NameMatches noteport.NameMatches
	Headings    noteport.Headings
	Summaries   noteport.VaultSummaries
	NoteTypes   noteport.NoteTypes
	TypedPaths  noteport.TypedPaths
	VaultHolds  noteport.VaultHolds
	NoteRefs    noteport.NoteRefs
	OpeningNote noteport.OpeningNote
	NamedPaths  noteport.NamedPaths
	Links       noteport.LinkQueries
	Problems    problemport.ProblemQueries
}

// IndexChunks are the questions the index answers about the chunks it holds, and their vectors.
type IndexChunks struct {
	Passages      sourceport.PassageQueries
	Nearest       sourceport.NearestQueries
	Progress      indexport.IndexProgress
	Stats         indexport.IndexStats
	Index         embeddingport.VectorIndex
	Vectors       embeddingport.VectorRepository
	VectorQueries embeddingport.VectorQueries
}

// IndexSources are the sources the index records, with the queries about them.
type IndexSources struct {
	Sources       sourceport.SourceRepository
	SourcesUnder  sourceport.SourcesUnder
	RemovalLookup sourceport.RemovalLookup
	SourceRecords sourceport.SourceRecords
	SourceBacklog sourceport.SourceBacklog
}

// IndexSearch is what the search of a vault ranks with and learns its floors from.
type IndexSearch struct {
	SearchFloors  searchpolicyport.FloorStore
	SearchSamples searchpolicyport.Samples
}

// IndexPorts are the ports and backend of an open index database, by what they answer.
type IndexPorts struct {
	Closer       io.Closer
	File         string
	Stores       IndexStores
	NoteQueries  IndexNoteQueries
	Chunks       IndexChunks
	Source       IndexSources
	Search       IndexSearch
	TextVersions textport.VersionStores
}

// Index exposes the cache as the ports the core requires: each group of them is a field a consumer
// reads what it needs from.
type Index struct {
	IndexPorts
	walks  Walks
	ledger *Ledger
}

// NewIndex constructs an Index over the provided IndexPorts.
func NewIndex(ports IndexPorts) *Index {
	return &Index{IndexPorts: ports, ledger: NewLedger()}
}

func (i *Index) Close() error { return i.Closer.Close() }

// Walks is the turns the vaults walked into this index take.
func (i *Index) Walks() *Walks { return &i.walks }

// Ledger returns the shared refresh ledger of this index.
func (i *Index) Ledger() *Ledger { return i.ledger }

// VectorIndex returns the store of vectors as a whole.
func (i *Index) VectorIndex() embeddingport.VectorIndex { return i.Chunks.Index }

// Path is where the index file is.
func (i *Index) Path() string { return i.File }
