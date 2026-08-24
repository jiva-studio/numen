package note

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	pathpkg "path"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
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
	Index   func(ctx context.Context, v domain.Vault, paths []string) error
	// Moving is told where the note went, so that whoever is showing it at the
	// name it had follows it. Nothing is told where nobody is drawing.
	Moving Moving
}

// Moved says where the note went and what it did to the links that pointed at
// it.
type Moved struct {
	From string
	To   string
	// Landed is whether the file is at To. It is false for a move that was
	// refused and for one that was not needed, and it stays true once the file
	// is there however the rest of the work goes.
	Landed bool
	// Repaired is the notes whose link stopped resolving and was written
	// again, by name.
	Repaired []string
	// Retargeted is the links that now resolve to a different note. They are
	// not broken and are not repaired: this is what two notes sharing a name
	// does, and it is the person's to settle.
	Retargeted []Retargeted
}

// Retargeted is one link that means something else now.
type Retargeted struct {
	// In is the note the link is written in; Target is where it goes; Now is
	// the note it currently reaches.
	In     string
	Target domain.Address
	Now    string
}

func (u Move) Execute(ctx context.Context, v domain.Vault, from, to string) (Moved, error) {
	res := Moved{From: from, To: to}
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
	if err := u.index(ctx, v, from, to); err != nil {
		return res, err
	}

	// The file is where it now is and the index is level with it. Whoever is
	// reading this note at the name it had is reading a name with no file.
	if u.Moving != nil {
		u.Moving(ctx, domain.Went{From: from, To: to})
	}

	name := domain.Basename(to)
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
			repaired, err := u.repair(ctx, v, was.From, was.Target, name)
			if err != nil {
				return res, err
			}
			if repaired {
				res.Repaired = append(res.Repaired, was.From)
			}
		default:
			res.Retargeted = append(res.Retargeted, Retargeted{
				In: was.From, Target: was.Target, Now: lands,
			})
		}
	}
	if len(res.Repaired) > 0 {
		if err := u.index(ctx, v, res.Repaired...); err != nil {
			return res, err
		}
	}
	return res, nil
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

// repair writes the name in place of an address that no longer reaches
// anything. Only the address changes, and only in the note that wrote it.
//
// It is a read and a write over a note somebody may have open, so it holds the
// vault's write lock across both.
func (u Move) repair(ctx context.Context, v domain.Vault, in string, address domain.Address, name string) (bool, error) {
	release, err := u.Writers.Hold(ctx, v)
	if err != nil {
		return false, err
	}
	defer release()

	reader, err := u.Readers.Open(v)
	if err != nil {
		return false, err
	}
	raw, err := reader.Read(ctx, in)
	if errors.Is(err, fs.ErrNotExist) {
		// The note that wrote the link went between the backlinks being read
		// and this. Its link went with it.
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %s: %w", in, err)
	}
	doc, err := markdown.Open(raw)
	if err != nil {
		// A note whose frontmatter cannot be read is never written.
		// Its link stays broken and is visible as a problem, which is the
		// honest outcome.
		return false, nil
	}
	inBlock, err := doc.PointLinksAt(address, name)
	if err != nil {
		return false, err
	}
	moved := inBlock + doc.PointProseAt(address, name)
	if moved == 0 {
		return false, nil
	}

	writer, err := u.Writers.Open(v)
	if err != nil {
		return false, err
	}
	_, err = writer.Write(ctx, in, doc.Bytes(), domain.FileRef{})
	return true, err
}

func (u Move) index(ctx context.Context, v domain.Vault, paths ...string) error {
	if u.Index == nil {
		return nil
	}
	return u.Index(ctx, v, paths)
}

// Into is where a note lands when it is filed under a folder, keeping its name.
func Into(folder, path string) string {
	return pathpkg.Join(folder, pathpkg.Base(path))
}
