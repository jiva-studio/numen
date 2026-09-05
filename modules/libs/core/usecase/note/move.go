package note

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	pathpkg "path"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Move files a note somewhere else, which is also how it is renamed: a note is
// named by the file it is in, so the two are one operation.
//
// The bytes do not change. A note that carried no identifier still carries
// none afterwards, because moving is not editing.
type Move struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	Links   port.LinkQueries
	// Names is asked how many notes are filed under a name, which is what says
	// whether a link repaired to that name reaches this note or another one.
	Names NameQueries
	// Sources is where the index files each file. A move tells it that what was
	// at one path is at another.
	Sources port.SourceRepository
	Index   Levels
	// Now is when this is happening. Bringing a note's title into line with its
	// filename is an edit, and an edit stamps the identifier a note arrived
	// without.
	Now port.Clock
	// Moving is told where the note went, so that whoever is showing it at the
	// name it had follows it. Nothing is told where nobody is drawing.
	Moving TellMove
	// Sync is asked, as each rename is made, whether a note's title and its
	// filename are kept as one name. Nothing asked keeps the two one name,
	// which is what an installation nobody has configured does.
	Sync SyncSetting
}

// NewMove is what files a note somewhere else: the vault it is read and written
// through, the links that point at it, what is asked which name reaches it,
// where the index files it, what brings the notes it repaired level, and what
// time it is.
//
// All seven are named here because a move short of any one of them lands the
// file and leaves something behind it — a link repaired to a bare name that
// reaches another note, a row still filed at the path the file left, a vault
// that cannot find what it now holds, or a title brought into line under an
// identifier minted off the machine's clock.
func NewMove(
	readers port.VaultReaders,
	writers port.VaultWriters,
	links port.LinkQueries,
	names NameQueries,
	sources port.SourceRepository,
	index Levels,
	now port.Clock,
) Move {
	return Move{
		Readers: readers, Writers: writers, Links: links,
		Names: names, Sources: sources, Index: index, Now: now,
	}
}

// MoveResult says where the note went and what it did to the links that
// pointed at it.
type MoveResult struct {
	From string
	To   string
	// Landed is whether the file is at To. It is false for a move that was
	// refused and for one that was not needed, and it stays true once the file
	// is there however the rest of the work goes.
	Landed bool
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
	pointing, err := u.Links.Backlinks(ctx, v.ID, from)
	if err != nil {
		return res, err
	}

	writer, err := u.Writers.Open(v)
	if err != nil {
		return res, err
	}
	if err := writer.Move(ctx, from, to); err != nil {
		return res, missing(err)
	}
	res.Landed = true
	if err := u.filed(ctx, v, from, to); err != nil {
		return res, err
	}
	return u.Settle(ctx, v, from, to, pointing)
}

// filed tells the index that what was at one path is at another. The bytes do
// not change, so nothing is read.
func (u Move) filed(ctx context.Context, v domain.Vault, from, to string) error {
	return u.Sources.MoveSources(ctx, v.ID, from, to)
}

// Settle is the work a move leaves once the file is where it was sent and the
// index is level with it: whoever is drawing the note told, and the links
// written by the name it had pointed at where it now is.
//
// Pointing is what pointed at the note before it went, read while there was
// still something to read.
func (u Move) Settle(ctx context.Context, v domain.Vault, from, to string, pointing []domain.ResolvedLink) (MoveResult, error) {
	res := MoveResult{From: from, To: to, Landed: true}

	// The file is where it now is and the index is level with it. Whoever is
	// reading this note at the name it had is reading a name with no file.
	if u.Moving != nil {
		u.Moving(ctx, domain.Move{From: from, To: to})
	}

	// What a repaired link is pointed at: the name the note is filed under, or
	// its path where that name is shared with another note.
	var address string
	if len(pointing) > 0 {
		var err error
		if address, err = u.addressed(ctx, v, to); err != nil {
			return res, err
		}
	}
	for _, was := range pointing {
		lands, still, err := u.landsOn(ctx, v, was)
		if err != nil {
			return res, err
		}
		if !still {
			// The note that wrote it has changed since, and the link is not in
			// it any more. There is nothing to repair and nothing to report.
			continue
		}
		switch lands {
		case to:
			// It followed the note, which is what a name does.
		case "":
			repaired, dangling, err := u.repair(ctx, v, was.From, was.Target, address)
			switch {
			case err != nil:
				return res, err
			case repaired:
				res.Repaired = append(res.Repaired, was.From)
			case dangling:
				res.Dangling = append(res.Dangling, was.From)
			}
		default:
			// It reaches another note of the same name. The link is not broken,
			// and which of the two a name means is the person's to settle.
		}
	}
	if len(res.Repaired) > 0 {
		if err := u.index(ctx, v, res.Repaired...); err != nil {
			return res, err
		}
	}
	return res, nil
}

// addressed is how a link reaches the note at this path, asked of the vault
// now that the file is there. A name no link reaches comes back as it stands,
// and the repair that writes it changes nothing.
func (u Move) addressed(ctx context.Context, v domain.Vault, path string) (string, error) {
	to, err := Addressed(ctx, u.Names, v.ID, path)
	if errors.Is(err, ErrUnaddressable) {
		return domain.Basename(path), nil
	}
	if err != nil {
		return "", err
	}
	return to.Value, nil
}

// landsOn is where one link goes now, asked of the note it is written in. A
// link resolves as of this moment and never as of when it was read.
//
// It reports whether the link is still written there at all. One that is not
// has been taken out since the backlinks were read, and saying where it used to
// go would report a move that nobody made.
func (u Move) landsOn(ctx context.Context, v domain.Vault, was domain.ResolvedLink) (lands string, still bool, err error) {
	links, err := u.Links.Links(ctx, v.ID, was.From)
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
) (repaired, dangling bool, err error) {
	release, err := u.Writers.Hold(ctx, v)
	if err != nil {
		return false, false, err
	}
	defer release()

	reader, err := u.Readers.Open(v)
	if err != nil {
		return false, false, err
	}
	raw, err := reader.Read(ctx, in)
	if errors.Is(err, fs.ErrNotExist) {
		// The note that wrote the link went between the backlinks being read
		// and this. Its link went with it.
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("read %s: %w", in, err)
	}
	doc, err := markdown.Open(raw)
	if err != nil {
		// A note whose frontmatter cannot be read is never written, and the
		// link it holds is left reaching nothing.
		return false, true, nil
	}
	inBlock, err := doc.PointLinksAt(address, reaches)
	if err != nil {
		return false, false, err
	}
	moved := inBlock + doc.PointProseAt(address, reaches)
	if moved == 0 {
		return false, false, nil
	}

	writer, err := u.Writers.Open(v)
	if err != nil {
		return false, false, err
	}
	_, err = writer.Write(ctx, in, doc.Bytes(), domain.Fingerprint{})
	return true, false, err
}

func (u Move) index(ctx context.Context, v domain.Vault, paths ...string) error {
	return Levelled(u.Index(ctx, v, paths), paths...)
}

// Into is where a note lands when it is filed under a folder, keeping its name.
func Into(folder, path string) string {
	return pathpkg.Join(folder, pathpkg.Base(path))
}
