package flashcards

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// MostRead is how many of the joined notes come back with their text.
//
// A hub note is pointed at by hundreds of others and a note may be a megabyte,
// so reading every one of them is a reading of the vault. The rest come back
// named and unread, and are read when somebody asks for one.
const MostRead = 30

// Joined is what a deck is joined to.
type Joined struct {
	Notes []Neighbour
	// Unread is how many at the end came without their text.
	Unread int
}

// Neighbour is one note the deck is joined to.
type Neighbour struct {
	// Written is the address as the deck wrote it, and all a dangling link
	// has. Empty for a note that points at the deck: what it wrote there names
	// the deck, not itself.
	Written string
	// Path is empty when the link resolves to nothing.
	Path  string
	Title string
	Body  string
	// Label is what the person called the relationship, where they did.
	Label string
	// Points is the deck pointing at this note; otherwise it points at the deck.
	Points bool
	// Ambiguous is several notes answering to the name that was written. The
	// link resolves to the nearest, and a person reading the wrong note has no
	// other way to find out.
	Ambiguous bool
	// Outcome is how the reading of the text ended. Empty for a note whose text
	// was never asked for.
	Outcome note.Outcome
}

// Around is what the deck being reviewed is joined to, with the text of each.
//
// A card's links are written in the deck's file, so a deck already points at
// everything its cards name: this is the reading beside the review.
type Around struct {
	Linked note.ShowLinks
	Notes  port.NoteQueries
	Reads  note.Read
}

// Execute gathers what the deck is joined to: what it points at first, then
// what points at it.
//
// An error is the index being out of reach. Whatever went wrong with one note —
// gone, too long, a file nothing can open — is an outcome on that note's own
// entry, and the rest of them are still read.
func (u Around) Execute(ctx context.Context, v domain.Vault, deck string) (Joined, error) {
	linked, err := u.Linked.Execute(ctx, v, deck)
	if err != nil {
		return Joined{}, err
	}

	// A deck naming itself, and a card naming the deck it stands in, are not
	// somewhere else to read.
	seen := map[string]bool{deck: true}

	var found []Neighbour
	for _, l := range linked.Links {
		// Only an address that names a note can name one that is missing. An
		// attachment and a web address are neither read here nor gone, and
		// drawing them as notes whose name has come loose says a vault has a
		// question in it where it has none.
		if l.Target.Scheme != domain.SchemeName && l.Target.Scheme != domain.SchemeNote {
			continue
		}
		one := Neighbour{
			Written:   l.Target.Written(),
			Path:      l.To,
			Label:     l.Label,
			Points:    true,
			Ambiguous: l.Ambiguous,
		}
		if l.To == "" {
			// Dangling: named by how it is written, and there is nothing to read.
			found = append(found, one)
			continue
		}
		if l.ToVault != v.ID || seen[l.To] {
			// A link into another vault has nothing here to read it with.
			continue
		}
		seen[l.To] = true
		found = append(found, one)
	}
	for _, l := range linked.Backlinks {
		if seen[l.From] {
			continue
		}
		seen[l.From] = true
		// Nothing is ambiguous on this side: the note shown is the one that
		// wrote the link, whatever its own name resolved through.
		found = append(found, Neighbour{Path: l.From, Label: l.Label})
	}

	paths := make([]string, 0, len(found))
	for _, one := range found {
		if one.Path != "" {
			paths = append(paths, one.Path)
		}
	}
	notes, err := u.Notes.Notes(ctx, v.ID, paths)
	if err != nil {
		return Joined{}, err
	}
	kinds, err := u.Notes.Types(ctx, v.ID, paths)
	if err != nil {
		return Joined{}, err
	}

	var out Joined
	for _, one := range found {
		if one.Path != "" {
			ref, isNote := notes[one.Path]
			if !isNote {
				// The index answered where the link landed and then no longer
				// held a note there: it is written while this reads it.
				continue
			}
			if kinds[one.Path] == domain.TypeStencil {
				// Every card names the stencil it is cut by, so a deck points at
				// its stencils. A stencil is how a card is laid out and not what
				// it was written from.
				continue
			}
			one.Title = ref.Title
		}
		out.Notes = append(out.Notes, one)
	}
	if len(out.Notes) > MostRead {
		out.Unread = len(out.Notes) - MostRead
	}

	for i := range out.Notes {
		if i == MostRead {
			break
		}
		if out.Notes[i].Path == "" {
			continue
		}
		contents, err := u.Reads.Execute(ctx, v, out.Notes[i].Path)
		if err != nil {
			// A caller that went away is not a vault with something wrong in it.
			if ctx.Err() != nil {
				return Joined{}, err
			}
			// One file that could not be opened is one entry with no text, not
			// a panel a person cannot read the rest of. Nothing is said about
			// why: what went wrong is the file's and not the note's, and every
			// outcome there is names something about the note.
			continue
		}
		out.Notes[i].Outcome = contents.Outcome
		out.Notes[i].Body = contents.Body
	}
	return out, nil
}
