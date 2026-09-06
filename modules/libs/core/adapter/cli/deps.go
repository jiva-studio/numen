package cli

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// Locations are the places a person may point this installation at, as they
// typed them. An empty one is the platform's own.
type Locations struct {
	Index      string
	Registry   string
	ServiceDir string
}

// Deps is what one run of the terminal works through: one opener for each
// command, and nothing opened until the command that needs it asks.
//
// They are openers rather than the things themselves because a command pays
// only for its own machinery. Listing vaults opens no index, a search opens no
// provider that fills one, and reading a scan opens no transcriber — and some
// of what stands behind these fetches a model over a network, which is minutes
// a person who typed `vault list` does not wait.
type Deps struct {
	// Vaults is the list of vaults this installation keeps and what the vault
	// commands work on. Every command opens it: what a person typed is a name,
	// a path or an identity, and the list is what turns it into a vault.
	Vaults func() (Vaults, error)

	// Scan opens the walk that brings the index level with one vault, the
	// reading of its books and the making of its vectors. rebuild reads every
	// file, whatever the index remembers.
	Scan func(ctx context.Context, v domain.Vault, rebuild bool) (Scan, error)

	// Search opens the search a question is answered by. trouble is where a
	// half of it that could not run says so.
	Search func(ctx context.Context, trouble port.Trouble) (Search, error)

	// Links opens what a note points at and what points at it.
	Links func(ctx context.Context) (Links, error)

	// Problems opens what a scan could not act on.
	Problems func(ctx context.Context) (Problems, error)

	// Recognise opens what reads a scanned document. fetching is called before
	// anything is waited for, where a model has to arrive first.
	Recognise func(ctx context.Context, v domain.Vault, fetching func()) (Recognise, error)

	// Transcribe opens what listens to a recording, and calls fetching the way
	// Recognise does.
	Transcribe func(ctx context.Context, v domain.Vault, fetching func()) (Transcribe, error)

	// ProofreadReading opens what puts a document's reading right, and
	// ProofreadTranscript what puts a recording's transcript right. They are
	// two openers because a file is one or the other and nothing opens for the
	// kind it is not.
	ProofreadReading    func(ctx context.Context, v domain.Vault) (ProofreadReading, error)
	ProofreadTranscript func(ctx context.Context, v domain.Vault) (ProofreadTranscript, error)
}

// Vaults is what the vault commands work on: the list this installation keeps,
// the folders it names, what gives a folder its identity, the place this
// machine keeps what a person deleted, and what time it is.
type Vaults struct {
	Registry port.VaultRegistry
	Readers  port.VaultReaders
	Identity port.VaultIdentity
	Trash    port.Trash
	Now      port.Clock

	// Rows opens what the index remembers about vaults, which a rename, a
	// forget and an erasure change together with the list. Listing vaults calls
	// it not at all, and opens no database.
	Rows func(ctx context.Context) (VaultRows, error)
}

// find resolves what the person typed against the list and, where it resolves
// to nothing, says what to do about it. Talking to a person belongs here.
func (v Vaults) find(nameOrPath string) (domain.Vault, error) {
	held, err := vault.NewFind(v.Registry).Execute(nameOrPath)
	if err != nil {
		return domain.Vault{}, fmt.Errorf("%w — add it with: numen-cli vault add %s", err, nameOrPath)
	}
	return held, nil
}

// VaultRows is what the index remembers about vaults, open.
type VaultRows struct {
	Vaults port.VaultRepository
	Close  func() error
}

// Scan is one vault made searchable.
type Scan struct {
	// Read is the walk, the books and the vectors, in the order they run in.
	Read vault.ReadWholeVault
	// Summary is what the index holds about the vault, asked once the walk is
	// over.
	Summary func(ctx context.Context) (domain.VaultSummary, error)
	// Unembedded is why no vectors are being made, and nothing where they are.
	Unembedded error
	Close      func() error
}

// Search is one question answered.
type Search struct {
	Search search.Search
	// Words is why the question is answered by its words alone, and nothing
	// where a model answers it too.
	Words error
	Close func() error
}

// Links is what one note points at and what points at it.
type Links struct {
	Show  note.ShowLinks
	Close func() error
}

// Problems is what a scan could not act on, as the index answers about it.
//
// The checks themselves are not opened here. Which questions are asked of a
// vault is the core's own vocabulary and no part of an installation, so the
// terminal takes the standard set and this opens what answers them.
type Problems struct {
	Problems port.ProblemQueries
	Close    func() error
}

// Recognise is one scanned document read with a model.
type Recognise struct {
	Recognise source.Recognise
	// Cut makes what a batch of pages wrote searchable before the next batch is
	// read, so a document stopped part way through answers to the page it
	// reached.
	Cut   source.Extract
	Close func() error
}

// Transcribe is one recording listened to with a model.
type Transcribe struct {
	Transcribe source.Transcribe
	// Cut is what it is for a reading: the speech already written down is
	// searchable while the rest is still being heard.
	Cut   source.Extract
	Close func() error
}

// ProofreadReading is one document's reading put right with a model.
type ProofreadReading struct {
	Proofread source.ProofreadReading
	Cut       source.Extract
	// Held says whether anything is configured to proofread with. Nothing else
	// here is opened where nothing is.
	Held  bool
	Close func() error
}

// ProofreadTranscript is one recording's transcript put right with a model.
type ProofreadTranscript struct {
	Proofread source.ProofreadTranscript
	Cut       source.Extract
	Held      bool
	Close     func() error
}

// closing gives back what an opener opened, and does nothing where it opened
// nothing.
func closing(close func() error) {
	if close != nil {
		_ = close()
	}
}
