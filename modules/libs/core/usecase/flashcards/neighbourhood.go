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

// Neighbourhood is what a deck is joined to.
type Neighbourhood struct {
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
	// Backlink is this note pointing at the deck; otherwise the deck points
	// at it.
	Backlink bool
	// Ambiguous is several notes answering to the name that was written. The
	// link resolves to the nearest, and a person reading the wrong note has no
	// other way to find out.
	Ambiguous bool
	// Outcome is how the reading of the text ended. Empty for a note whose text
	// was never asked for.
	Outcome note.ReadOutcome
}

// ShowNeighbourhood is what the deck being reviewed is joined to, with the text
// of each.
//
// A card's links are written in the deck's file, so a deck already points at
// everything its cards name: this is the reading beside the review.
type ShowNeighbourhood struct {
	Linked note.ShowLinks
	Notes  port.NoteQueries
	Reads  note.Read
}

// NewShowNeighbourhood is what the reading beside a review is made of: what the
// deck is joined to, what says which of those notes are decks and stencils of
// its own, and what the prose of each is read out of.
//
// All three are named here because the last is reached only once a deck names
// something: a build short of it opens every deck that points nowhere and goes
// down on the first one a person wrote a link into.
func NewShowNeighbourhood(
	linked note.ShowLinks, notes port.NoteQueries, reads note.Read,
) ShowNeighbourhood {
	return ShowNeighbourhood{Linked: linked, Notes: notes, Reads: reads}
}

// Execute gathers what the deck is joined to: what it points at first, then
// what points at it.
//
// An error is the vault or the index being out of reach. What is wrong with one
// note — gone, too long, not a note at all — is an outcome on that note's own
// entry, and the rest of them are still read.
func (u ShowNeighbourhood) Execute(
	ctx context.Context, v domain.Vault, deck string,
) (Neighbourhood, error) {
	linked, err := u.Linked.Execute(ctx, v, deck)
	if err != nil {
		return Neighbourhood{}, err
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
		found = append(found, Neighbour{Path: l.From, Label: l.Label, Backlink: true})
	}

	paths := make([]string, 0, len(found))
	for _, one := range found {
		if one.Path != "" {
			paths = append(paths, one.Path)
		}
	}
	notes, err := u.Notes.Notes(ctx, v.ID, paths)
	if err != nil {
		return Neighbourhood{}, err
	}
	kinds, err := u.Notes.Types(ctx, v.ID, paths)
	if err != nil {
		return Neighbourhood{}, err
	}

	var out Neighbourhood
	for _, one := range found {
		if one.Path != "" {
			ref, isNote := notes[one.Path]
			if !isNote {
				// The index answered where the link landed and then no longer
				// held a note there: it is written while this reads it.
				continue
			}
			// An ordinary note is what the cards were written from, and is
			// taken below rather than passed over here.
			//exhaustive:ignore
			switch kinds[one.Path] {
			case domain.TypeStencil:
				// Every card names the stencil it is cut by, so a deck points at
				// its stencils. A stencil is how a card is laid out and not what
				// it was written from.
				continue
			case domain.TypePreset:
				// A deck points at the preset it is scheduled by. A preset is how
				// the cards come round and not what they were written from.
				continue
			case domain.TypeDeck:
				// Another deck is more cards to answer, and this is what the cards
				// in front of a person were written from.
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
			// A read fails when the vault itself is out of reach, and a panel
			// of titles with no prose under any of them would say nothing about
			// why. What is wrong with one note is an outcome, not an error.
			return Neighbourhood{}, err
		}
		out.Notes[i].Outcome = contents.Outcome
		out.Notes[i].Body = contents.Body
	}
	return out, nil
}
