package note

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
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
	// names is asked how many notes are filed under a name, which is what says
	// whether a link repaired to that name reaches this note or another one.
	names NameQueries
	// sources is where the index files each file. A move tells it that what was
	// at one path is at another.
	sources sourceport.SourceRepository
	index   Levels
	// now is when this is happening. Bringing a note's title into line with its
	// filename is an edit, and an edit stamps the identifier a note arrived
	// without.
	now machineport.Clock
	// Drawing is told where the note went, so that whoever is showing it at the
	// name it had follows it. Nothing is told where nobody is drawing.
	Drawing TellMove
	// Sync is asked, as each rename is made, whether a note's title and its
	// filename are kept as one name. Nothing asked keeps the two one name,
	// which is what an installation nobody has configured does.
	Sync SyncSetting
}

// NewMove is what files a note somewhere else: the vault it is read and written
// through, the links that point at it, what is asked which name reaches it,
// where the index files it, what brings the notes it repaired level, and what
// time it is.
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

// MoveResult says where the note went and what it did to the links that
// pointed at it.
type MoveResult struct {
	From string
	To   string
	// IsLanded is whether the file is at To. It is false for a move that was
	// is there however the rest of the work goes.
	IsLanded bool
	// Repaired is the notes whose link stopped resolving and was written
	// again, by name.
	Repaired []string
	// Dangling is the notes whose link stopped resolving and could not be
	// written again. Each of them still points at the name the file left.
	Dangling []string
}

func (u Move) Execute(ctx context.Context, v domain.Vault, from, to string) (MoveResult, error) {
	res := MoveResult{From: from, To: to}
	if from == to {
		return res, nil
	}

	// Asked before the move, because afterwards nothing points at the old path
	// and there is nothing left to ask about.
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

// moveInIndex tells the index that what was at one path is at another. The
// bytes do not change, so nothing is read.
func (u Move) moveInIndex(ctx context.Context, v domain.Vault, from, to string) error {
	return u.sources.MoveSources(ctx, v.ID, from, to)
}

// RepairLinks is the work a move leaves once the file is where it was sent and the
// index is level with it: whoever is drawing the note told, and the links
// written by the name it had pointed at where it now is.
//
// Pointing is what pointed at the note before it went, read while there was
// still something to read.
func (u Move) RepairLinks(ctx context.Context, v domain.Vault, from, to string, pointing []domain.ResolvedLink) (MoveResult, error) {
	res := MoveResult{From: from, To: to, IsLanded: true}

	// The file is where it now is and the index is level with it. Whoever is
	// reading this note at the name it had is reading a name with no file.
	if u.Drawing != nil {
		u.Drawing(ctx, domain.Move{From: from, To: to})
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

// rewriteLink writes one link that pointed at the note again where it reaches
// nothing, and records the note that wrote it in res.
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
	// A link that lands on the note followed it, which is what a name does. One
	// that lands on another note of the same name is not broken, and which of the
	// two a name means is the person's to settle.
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

// getAddress is how a link reaches the note at this path, asked of the vault
// now that the file is there. A name no link reaches comes back empty,
// and the repair that writes it changes nothing.
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

// getLanding is where one link goes now, asked of the note it is written in. A
// link resolves as of this moment and never as of when it was read.
//
// It reports whether the link is still written there at all. One that is not
// has been taken out since the backlinks were read, and saying where it used to
// go would report a move that nobody made.
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

// repair writes `reaches` in place of an address that no longer reaches
// anything. Only the address changes, and only in the note that wrote it.
//
// It says whether the link was written again and, where it was not, whether the
// note still holds it: a note nothing could be written into keeps a link that
// reaches nothing, and the caller has to be able to name it.
//
// It is a read and a write over a note somebody may have open, so it holds the
// vault's write lock across both.
func (u Move) repair(
	ctx context.Context, v domain.Vault, in string, address domain.Address, reaches string,
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
	if errors.Is(err, fs.ErrNotExist) {
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
	inBlock, err := doc.RedirectLinks(address, reaches)
	if err != nil {
		return linkUntouched, err
	}
	inProse, err := doc.RedirectProse(address, reaches)
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
	if _, err := writer.Write(ctx, in, doc.Bytes(), domain.Fingerprint{}); err != nil {
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
