package filetext

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// sourcesPerQuery is the page size of the query for sources that need text,
// so a library of any size is processed in pages.
const sourcesPerQuery = 100

// Index cuts the live text versions of the files a vault holds into chunks, in two passes.
// The first records the files that have text or that a plugin can extract text
// from, without opening them. The second reads the live text of every file that
// needs it and cuts it into chunks, one write per file.
type Index struct {
	readers vaultport.VaultReaders
	sources sourceport.SourceRepository
	queries sourceQueries
	texts   text.Readers

	Sizes chunking.Sizes

	Legibility chunking.Legibility

	ShouldRebuildIndex bool

	errorHandler window.ErrorHandler
	readNote     func(ctx context.Context, v domain.Vault, path string) error
}

// NewIndex creates an Index from the vault readers, the source repository
// that stores sources and chunks, the queries that report which sources the
// index records, which still need text and the text kept for the ones that left,
// and the text readers. Nil texts means no file has text. The result ignores the errors a pass reports and reads no note again before it is cut;
// WithHandlers changes both.
func NewIndex(
	readers vaultport.VaultReaders,
	sources sourceport.SourceRepository,
	records sourceport.SourceRecords,
	backlog sourceport.SourceBacklog,
	lookup sourceport.RemovalLookup,
	texts text.Readers,
) Index {
	return Index{
		readers: readers, sources: sources, texts: texts,
		queries: sourceQueries{records: records, backlog: backlog, lookup: lookup},
	}
}

// WithHandlers returns the Index that passes the errors a pass reports to errorHandler and calls readNote
// to read a note again before it is cut. A nil errorHandler or readNote does neither.
func (u Index) WithHandlers(errorHandler window.ErrorHandler, readNote func(ctx context.Context, v domain.Vault, path string) error) Index {
	u.errorHandler, u.readNote = errorHandler, readNote
	return u
}

// IndexWithProgress runs Execute with onProgress reporting each step when
// non-nil, and leaves uncut every source isSkipped names when non-nil.
func (u Index) IndexWithProgress(
	ctx context.Context, v domain.Vault, onProgress func(vaultreading.IndexResult), isSkipped func(path string) bool,
) (vaultreading.IndexResult, error) {
	return u.run(ctx, v, onProgress, isSkipped)
}

// Execute brings the chunks of one vault's file texts up to date with what is
// on disk.
func (u Index) Execute(ctx context.Context, v domain.Vault) (vaultreading.IndexResult, error) {
	return u.run(ctx, v, nil, nil)
}

func (u Index) run(
	ctx context.Context, v domain.Vault, onProgress func(vaultreading.IndexResult), isSkipped func(path string) bool,
) (vaultreading.IndexResult, error) {
	var res vaultreading.IndexResult

	on, err := u.open(v)
	if err != nil {
		return res, err
	}
	noteStats, notePaths, err := newDiscoverer(u).discover(ctx, v, on, &res)
	if err != nil {
		return res, err
	}
	notePaths, err = newOutdated(u).dropOutdatedText(ctx, v, on, notePaths, &res)
	if err != nil {
		return res, err
	}
	return res, newCutter(u, onProgress, isSkipped).cut(ctx, v, on, noteStats, notePaths, &res)
}

// IndexOne brings a single file's chunks up to date with its live text
// version. It is what the index does when the versions of a file change.
func (u Index) IndexOne(ctx context.Context, v domain.Vault, path string) (vaultreading.IndexResult, error) {
	var res vaultreading.IndexResult
	on, err := u.open(v)
	if err != nil {
		return res, err
	}
	return res, newCutter(u, nil, nil).source(ctx, v, on, path, u.sizes(), &res)
}

// openVault is one vault as a cut reads it: its files and the text versions of them.
type openVault struct {
	files vaultport.VaultReader
	texts text.VaultTexts
}

func (u Index) open(v domain.Vault) (openVault, error) {
	files, err := u.readers.Open(v)
	if err != nil {
		return openVault{}, err
	}
	if u.texts == nil {
		return openVault{files: files, texts: noTexts{files: files}}, nil
	}
	texts, err := u.texts.Open(v)
	if err != nil {
		return openVault{}, err
	}
	return openVault{files: files, texts: texts}, nil
}

// noTexts is a vault with no text versions: every file has no text, and no
// text is requested.
type noTexts struct {
	files vaultport.VaultReader
}

// GetContentHash hashes the file from a stream on every call: with no versions there is no record to keep.
func (t noTexts) GetContentHash(ctx context.Context, path string) (_ string, err error) {
	file, err := t.files.Open(ctx, path)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	return domain.HashReader(file)
}

func (noTexts) FindLiveVersion(context.Context, string, string) (domain.TextVersionID, error) {
	return "", text.ErrNoText
}

func (noTexts) GetLiveText(context.Context, string, string) (domain.VersionText, error) {
	return domain.VersionText{}, text.ErrNoText
}

func (noTexts) GetVersionText(context.Context, string, domain.TextVersionID) (domain.VersionText, error) {
	return domain.VersionText{}, text.ErrNoText
}

func (noTexts) IsTextSource(context.Context, string) (bool, error) { return false, nil }

// Recipe returns the recipe that names the version reader and the chunk sizes.
func (u Index) Recipe() string { return chunking.GetTextVersionRecipe(u.sizes()) }

func (u Index) sizes() chunking.Sizes { return u.Sizes.Resolve() }
