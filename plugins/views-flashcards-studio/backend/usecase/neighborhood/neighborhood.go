// Package neighborhood reads what a deck is joined to: the notes its cards link to and the notes
// that link to it, each with its title and text.
package neighborhood

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// MostRead is how many of the joined notes are returned with their text.
//
// A hub note is pointed at by hundreds of others and a note may be a megabyte,
// so reading every one reads the whole vault. The remainder is returned named and
// unread, and is read on request.
const MostRead = 30

// Neighborhood is what a deck is joined to.
type Neighborhood struct {
	Notes  []Neighbor
	Unread int
}

// Neighbor is one note the deck is joined to.
type Neighbor struct {
	Written     string
	Path        string
	Title       string
	Body        string
	Label       string
	IsBacklink  bool
	IsAmbiguous bool
	Outcome     edit.ReadOutcome
}

// neighborNotes is what the neighborhood asks the index of the notes it found.
type neighborNotes interface {
	ports.NoteRefQueries
	ports.NoteTypeQueries
}

// linkReader returns what a note points at and what points at it.
type linkReader interface {
	Execute(ctx context.Context, v domain.Vault, path string) (format.NoteLinks, error)
}

// noteReader reads the prose of one note. What is wrong with the note itself is an outcome.
type noteReader interface {
	Execute(ctx context.Context, v domain.Vault, path string) (edit.Contents, error)
}

// Show is what the deck being reviewed is joined to, with the text
// of each.
//
// A card's links are written in the deck's file, so a deck already points at
// everything its cards name: this is the reading beside the review.
type Show struct {
	linked linkReader
	notes  neighborNotes
	reads  noteReader
}

// NewShow constructs a Show from the source of the
// deck's links, the note queries that identify which of those notes are decks and
// card models, and the reader of each note's prose.
//
// The prose is read only where a deck names something.
func NewShow(
	linked linkReader, notes neighborNotes, reads noteReader,
) Show {
	return Show{linked: linked, notes: notes, reads: reads}
}

// Execute gathers what the deck is joined to: what it points at first, then what points at it.
// An error is the vault or the index being out of reach. What is wrong with one note (gone, too
// long, not a note) is an outcome on that note's own entry, and the rest are still read.
func (u Show) Execute(
	ctx context.Context, v domain.Vault, deck string,
) (Neighborhood, error) {
	linked, err := u.linked.Execute(ctx, v, deck)
	if err != nil {
		return Neighborhood{}, err
	}

	found := getNeighbors(v, deck, linked)
	paths := make([]string, 0, len(found))
	for _, one := range found {
		if one.Path != "" {
			paths = append(paths, one.Path)
		}
	}
	notes, err := u.notes.GetNotes(ctx, v.ID, paths)
	if err != nil {
		return Neighborhood{}, err
	}
	kinds, err := u.notes.GetTypes(ctx, v.ID, paths)
	if err != nil {
		return Neighborhood{}, err
	}

	var out Neighborhood
	for _, one := range found {
		titled, isShown := getNeighborTitle(one, notes, kinds)
		if isShown {
			out.Notes = append(out.Notes, titled)
		}
	}
	if len(out.Notes) > MostRead {
		out.Unread = len(out.Notes) - MostRead
	}
	if err := u.readBodies(ctx, v, out.Notes); err != nil {
		return Neighborhood{}, err
	}
	return out, nil
}

func getNeighbors(v domain.Vault, deck string, linked format.NoteLinks) []Neighbor {
	observed := map[string]bool{deck: true}
	var found []Neighbor
	for _, l := range linked.Links {
		one, isKept := getLinkedNeighbor(v, l, observed)
		if isKept {
			found = append(found, one)
		}
	}
	for _, l := range linked.Backlinks {
		if observed[l.From] {
			continue
		}
		observed[l.From] = true
		// Nothing is ambiguous on this side: the note shown is the one that
		// wrote the link, whatever its own name resolved through.
		found = append(found, Neighbor{Written: "", Path: l.From, Title: "", Body: "", Label: l.Label, IsBacklink: true, IsAmbiguous: false, Outcome: ""})
	}
	return found
}

func getLinkedNeighbor(v domain.Vault, l domain.ResolvedLink, visited map[string]bool) (Neighbor, bool) {
	// Only an address that names a note can name a missing note. Attachments and web
	// addresses are neither read here nor missing, and drawing them as notes with
	// unresolved names would report a vault problem that does not exist.
	if l.Target.Scheme != domain.SchemeName && l.Target.Scheme != domain.SchemeNote {
		// gomutants:disable-next-line RETURN_ZERO reason="Struct literal already holds the zero Neighbor value."
		return Neighbor{Written: "", Path: "", Title: "", Body: "", Label: "", IsBacklink: false, IsAmbiguous: false, Outcome: ""}, false
	}
	one := Neighbor{
		Written:     l.Target.GetWritten(),
		Path:        l.To,
		Title:       "",
		Body:        "",
		Label:       l.Label,
		IsBacklink:  false,
		IsAmbiguous: l.IsAmbiguous,
		Outcome:     "",
	}
	if l.To == "" {
		// Dangling: named by how it is written, and there is nothing to read.
		return one, true
	}
	if l.ToVault != v.ID || visited[l.To] {
		// A link into another vault has nothing here to read it with.
		// gomutants:disable-next-line RETURN_ZERO reason="Struct literal already holds the zero Neighbor value."
		return Neighbor{Written: "", Path: "", Title: "", Body: "", Label: "", IsBacklink: false, IsAmbiguous: false, Outcome: ""}, false
	}
	visited[l.To] = true
	return one, true
}

func getNeighborTitle(
	one Neighbor, notes map[string]domain.NoteRef, kinds map[string]domain.NoteType,
) (Neighbor, bool) {
	if one.Path == "" {
		return one, true
	}
	ref, isNote := notes[one.Path]
	if !isNote {
		// The index reported where the link lands but holds no note there: the entry
		// changed during this read.
		// gomutants:disable-next-line RETURN_ZERO reason="Struct literal already holds the zero Neighbor value."
		return Neighbor{Written: "", Path: "", Title: "", Body: "", Label: "", IsBacklink: false, IsAmbiguous: false, Outcome: ""}, false
	}
	// An ordinary note is what the cards were written from, and is
	// taken below.
	//exhaustive:ignore
	switch kinds[one.Path] {
	case format.TypeCardModel:
		// Every card names the card model it is cut by, so a deck points at
		// its card models. A card model is how a card is laid out and not what
		// it was written from.
		// gomutants:disable-next-line RETURN_ZERO reason="Struct literal already holds the zero Neighbor value."
		return Neighbor{Written: "", Path: "", Title: "", Body: "", Label: "", IsBacklink: false, IsAmbiguous: false, Outcome: ""}, false
	case format.TypePreset:
		// A deck points at the preset it is scheduled by. A preset is how
		// the cards come round and not what they were written from.
		// gomutants:disable-next-line RETURN_ZERO reason="Struct literal already holds the zero Neighbor value."
		return Neighbor{Written: "", Path: "", Title: "", Body: "", Label: "", IsBacklink: false, IsAmbiguous: false, Outcome: ""}, false
	case format.TypeDeck:
		// Another deck holds more cards to answer and is not a source of the cards under
		// review.
		// gomutants:disable-next-line RETURN_ZERO reason="Struct literal already holds the zero Neighbor value."
		return Neighbor{Written: "", Path: "", Title: "", Body: "", Label: "", IsBacklink: false, IsAmbiguous: false, Outcome: ""}, false
	}
	one.Title = ref.Title
	return one, true
}

func (u Show) readBodies(ctx context.Context, v domain.Vault, neighbors []Neighbor) error {
	for i := range neighbors {
		if i == MostRead {
			break
		}
		if neighbors[i].Path == "" {
			continue
		}
		contents, err := u.reads.Execute(ctx, v, neighbors[i].Path)
		if err != nil {
			return err
		}
		neighbors[i].Outcome = contents.Outcome
		neighbors[i].Body = contents.Body
	}
	return nil
}
