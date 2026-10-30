package note

import (
	"context"
	"errors"
	"fmt"
	pathpkg "path"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// Move files a note somewhere else, which is also how it is renamed: a note is
// named by the file it is in, so the two are one operation.
//
// The bytes do not change. A note that carried no identifier still carries
// none afterwards, because moving is not editing.
type Move struct {
	readers vaultport.VaultReaders
	writers vaultport.VaultWriters
	links   noteport.LinkQueries
	// names counts the notes filed under a name, which decides whether a repaired
	// link to that name resolves to this note or to another one.
	names NameQueries
	// sources is the index's file registry. A move records that the file at one
	// path is now at another.
	sources sourceport.SourceRepository
	index   Levels
	// now is when this is happening. Bringing a note's title into line with its
	// filename is an edit, and an edit stamps the identifier a note arrived
	// without.
	now machineport.Clock
	// Drawing receives the note's new path; every view showing the note
	// under its old name follows it. Nil sends no notification.
	Drawing TellMove
	// Sync reports, for each rename, whether a note's title and filename are kept
	// identical. Nil keeps them identical.
	Sync SyncSetting
}

// NewMove returns a Move that files a note at another path. It takes the vault
// readers and writers, the link queries that find links to the note, the name
// queries that decide which name resolves to it, the index file registry, the
// level that syncs the repaired notes with the index, and the clock.
func NewMove(
	readers vaultport.VaultReaders,
	writers vaultport.VaultWriters,
	links noteport.LinkQueries,
	names NameQueries,
	sources sourceport.SourceRepository,
	index Levels,
	now machineport.Clock,
) Move {
	return Move{
		readers: readers, writers: writers, links: links,
		names: names, sources: sources, index: index, now: now,
	}
}

// WithSources replaces the source repository.
func (u Move) WithSources(sources sourceport.SourceRepository) Move {
	u.sources = sources
	return u
}

// WithReaders replaces the vault readers.
func (u Move) WithReaders(readers vaultport.VaultReaders) Move {
	u.readers = readers
	return u
}

// WithWriters replaces the vault writers.
func (u Move) WithWriters(writers vaultport.VaultWriters) Move {
	u.writers = writers
	return u
}

// Sources returns the source repository.
func (u Move) Sources() sourceport.SourceRepository { return u.sources }

// Writers returns the vault writers.
func (u Move) Writers() vaultport.VaultWriters { return u.writers }

// Readers returns the vault readers.
func (u Move) Readers() vaultport.VaultReaders { return u.readers }

// MoveResult reports the new path of the note and the changes made to the links
// that pointed at it.
type MoveResult = domain.MoveResult

func (u Move) Execute(ctx context.Context, v domain.Vault, from, to string) (MoveResult, error) {
	res := MoveResult{From: from, To: to}
	if from == to {
		return res, nil
	}

	// Backlinks are read before the move; afterwards nothing points at the old
	// path.
	pointing, err := u.links.GetBacklinks(ctx, v.ID, from)
	if err != nil {
		return res, err
	}

	writer, err := u.writers.Open(v)
	if err != nil {
		return res, err
	}
	if err := writer.Move(ctx, from, to); err != nil {
		return res, mapMissingNote(err)
	}
	res.IsLanded = true
	if err := u.moveInIndex(ctx, v, from, to); err != nil {
		return res, err
	}
	return u.RepairLinks(ctx, v, from, to, pointing)
}

// moveInIndex records in the index that the file at one path is now at
// another. The bytes do not change, so nothing is read.
func (u Move) moveInIndex(ctx context.Context, v domain.Vault, from, to string) error {
	return u.sources.MoveSources(ctx, v.ID, from, to)
}

// RepairLinks completes a move after the file is at its new path and the index
// is in sync: it notifies the views of the note and rewrites the links that
// used the old name.
//
// pointing holds the links that resolved to the note before the move.
func (u Move) RepairLinks(ctx context.Context, v domain.Vault, from, to string, pointing []domain.ResolvedLink) (MoveResult, error) {
	res := MoveResult{From: from, To: to, IsLanded: true}

	// The file is at its new path and the index is in sync. A view still showing
	// the old name refers to a path with no file.
	if u.Drawing != nil {
		if err := u.Drawing(ctx, domain.Move{From: from, To: to}); err != nil {
			return res, fmt.Errorf("tell the move of %s to %s: %w", from, to, err)
		}
	}

	// What a repaired link is pointed at: the name the note is filed under, or
	// its path where that name is shared with another note.
	var address string
	if len(pointing) > 0 {
		var err error
		if address, err = u.getAddress(ctx, v, to); err != nil {
			return res, err
		}
	}
	for _, was := range pointing {
		if err := u.rewriteLink(ctx, v, address, was, &res); err != nil {
			return res, err
		}
	}
	if len(res.Repaired) > 0 {
		if err := u.level(ctx, v, res.Repaired...); err != nil {
			return res, err
		}
	}
	return res, nil
}

// rewriteLink rewrites one link that does not resolve and records the note
// containing it in res.
func (u Move) rewriteLink(
	ctx context.Context, v domain.Vault, address string, was domain.ResolvedLink, res *MoveResult,
) error {
	lands, still, err := u.getLanding(ctx, v, was)
	if err != nil {
		return err
	}
	if !still {
		// The note that wrote it has changed since, and the link is not in
		// it any more. There is nothing to repair and nothing to report.
		return nil
	}
	// A link that resolves to the moved note followed it. A link that resolves to
	// another note of the same name is not broken; which note a name denotes is the
	// user's decision.
	if lands != "" {
		return nil
	}
	remaining, err := u.repair(ctx, v, was.From, was.Target, address)
	switch {
	case err != nil:
		return err
	case remaining == linkRepaired:
		res.Repaired = append(res.Repaired, was.From)
	case remaining == linkDangling:
		res.Dangling = append(res.Dangling, was.From)
	}
	return nil
}

// getAddress returns the address by which a link resolves to the note at path,
// as the vault resolves it now that the file is there. An empty result means no
// link resolves to the note, and the repair changes nothing.
func (u Move) getAddress(ctx context.Context, v domain.Vault, path string) (string, error) {
	to, err := GetAddress(ctx, u.names, v.ID, path)
	if errors.Is(err, ErrUnaddressable) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return to.Value, nil
}

// getLanding returns where one link resolves now, evaluated in the note that
// contains it. Resolution is always current, never the state at the time the
// backlinks were read.
//
// exists reports whether the link is still present in that note. A link removed
// since the backlinks were read must not be reported as moved.
func (u Move) getLanding(ctx context.Context, v domain.Vault, was domain.ResolvedLink) (lands string, exists bool, err error) {
	links, err := u.links.GetLinks(ctx, v.ID, was.From)
	if err != nil {
		return "", false, err
	}
	for _, l := range links {
		if l.Target == was.Target {
			return l.To, true, nil
		}
	}
	return "", false, nil
}

// linkState is what became of a link that stopped resolving.
type linkState int

const (
	// linkUntouched is a link nothing was written for.
	linkUntouched linkState = iota
	// linkRepaired is a link written again.
	linkRepaired
	// linkDangling is a link left reaching nothing in a note that could not be
	// written.
	linkDangling
)

// repair writes `destination` over an address that does not resolve. Only
// the address changes, and only in the note containing it.
//
// It reports whether the link was rewritten and, if not, whether the note still
// contains it: when the note cannot be written, the unresolved link remains and
// the caller must be able to name it.
//
// It reads and writes a note that may be open elsewhere, so it holds the vault
// write lock across both.
func (u Move) repair(
	ctx context.Context, v domain.Vault, in string, address domain.Address, destination string,
) (linkState, error) {
	release, err := u.writers.Lock(ctx, v)
	if err != nil {
		return linkUntouched, err
	}
	defer release()

	reader, err := u.readers.Open(v)
	if err != nil {
		return linkUntouched, err
	}
	raw, err := reader.Read(ctx, in)
	if errors.Is(err, vaultport.ErrNotExist) {
		// The note that wrote the link went between the backlinks being read
		// and this. Its link went with it.
		return linkUntouched, nil
	}
	if err != nil {
		return linkUntouched, fmt.Errorf("read %s: %w", in, err)
	}
	doc, err := markdown.Open(raw)
	if err != nil {
		// A note whose frontmatter cannot be read is never written, and the
		// link it holds is left reaching nothing.
		return linkDangling, nil
	}
	inBlock, err := doc.RedirectLinks(address, destination)
	if err != nil {
		return linkUntouched, err
	}
	inProse, err := doc.RedirectProse(address, destination)
	if err != nil {
		return linkUntouched, err
	}
	moved := inBlock + inProse
	if moved == 0 {
		return linkUntouched, nil
	}

	writer, err := u.writers.Open(v)
	if err != nil {
		return linkUntouched, err
	}
	if _, err := writer.Write(ctx, in, doc.Bytes(), domain.FileStat{}); err != nil {
		return linkUntouched, err
	}
	return linkRepaired, nil
}

func (u Move) level(ctx context.Context, v domain.Vault, paths ...string) error {
	if u.index == nil {
		return nil
	}
	return WrapUnlevelled(u.index(ctx, v, paths), paths...)
}

// GetPathUnder is where a note lands when it is filed under a folder, keeping
// its name.
func GetPathUnder(folder, path string) string {
	return pathpkg.Join(folder, pathpkg.Base(path))
}
