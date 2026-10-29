package flashcards

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// MostRead is how many of the joined notes come back with their text.
//
// A hub note is pointed at by hundreds of others and a note may be a megabyte,
// so reading every one reads the whole vault. The remainder is returned named and
// unread, and is read on request.
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
	// Label is the relationship label given in the link, when one was given.
	Label string
	// IsBacklink is this note pointing at the deck. False is the deck pointing
	// at it.
	IsBacklink bool
	// IsAmbiguous is whether several notes match the name that was written. The link
	// resolves to the nearest match; this flag is the only signal that another note
	// shares the name.
	IsAmbiguous bool
	// Outcome is how reading the text ended. Empty for a note whose text was not
	// requested.
	Outcome note.ReadOutcome
}

// ShowNeighbourhood is what the deck being reviewed is joined to, with the text
// of each.
//
// A card's links are written in the deck's file, so a deck already points at
// everything its cards name: this is the reading beside the review.
type ShowNeighbourhood struct {
	linked note.ShowLinks
	notes  ports.NoteQueries
	reads  note.Read
}

// NewShowNeighbourhood constructs a ShowNeighbourhood from the source of the
// deck's links, the note queries that identify which of those notes are decks and
// card models, and the reader of each note's prose.
//
// The prose is read only where a deck names something.
func NewShowNeighbourhood(
	linked note.ShowLinks, notes ports.NoteQueries, reads note.Read,
) ShowNeighbourhood {
	return ShowNeighbourhood{linked: linked, notes: notes, reads: reads}
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
	linked, err := u.linked.Execute(ctx, v, deck)
	if err != nil {
		return Neighbourhood{}, err
	}

	found := getNeighbours(v, deck, linked)
	paths := make([]string, 0, len(found))
	for _, one := range found {
		if one.Path != "" {
			paths = append(paths, one.Path)
		}
	}
	notes, err := u.notes.GetNotes(ctx, v.ID, paths)
	if err != nil {
		return Neighbourhood{}, err
	}
	kinds, err := u.notes.GetTypes(ctx, v.ID, paths)
	if err != nil {
		return Neighbourhood{}, err
	}

	var out Neighbourhood
	for _, one := range found {
		titled, isShown := getNeighbourTitle(one, notes, kinds)
		if isShown {
			out.Notes = append(out.Notes, titled)
		}
	}
	if len(out.Notes) > MostRead {
		out.Unread = len(out.Notes) - MostRead
	}
	if err := u.readBodies(ctx, v, out.Notes); err != nil {
		return Neighbourhood{}, err
	}
	return out, nil
}

// getNeighbours is what the links of a deck lead to: what it points at first,
// then what points at it. A deck naming itself, and a card naming the deck it
func getNeighbours(v domain.Vault, deck string, linked note.Links) []Neighbour {
	observed := map[string]bool{deck: true}
	var found []Neighbour
	for _, l := range linked.Links {
		one, isKept := getLinkedNeighbour(v, l, observed)
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
		found = append(found, Neighbour{Written: "", Path: l.From, Title: "", Body: "", Label: l.Label, IsBacklink: true, IsAmbiguous: false, Outcome: ""})
	}
	return found
}

// getLinkedNeighbour is the neighbour a link of the deck leads to, and whether
// it is one to show. A note reached is marked in seen.
func getLinkedNeighbour(v domain.Vault, l domain.ResolvedLink, visited map[string]bool) (Neighbour, bool) {
	// Only an address that names a note can name a missing note. Attachments and web
	// addresses are neither read here nor missing, and drawing them as notes with
	// unresolved names would report a vault problem that does not exist.
	if l.Target.Scheme != domain.SchemeName && l.Target.Scheme != domain.SchemeNote {
		return Neighbour{Written: "", Path: "", Title: "", Body: "", Label: "", IsBacklink: false, IsAmbiguous: false, Outcome: ""}, false
	}
	one := Neighbour{
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
		return Neighbour{Written: "", Path: "", Title: "", Body: "", Label: "", IsBacklink: false, IsAmbiguous: false, Outcome: ""}, false
	}
	visited[l.To] = true
	return one, true
}

// getNeighbourTitle is the neighbour with its title, and whether it is one to show.
// A neighbour with a path is shown when the index still holds a note there and
// the note is one the cards were written from.
func getNeighbourTitle(
	one Neighbour, notes map[string]domain.NoteRef, kinds map[string]domain.NoteType,
) (Neighbour, bool) {
	if one.Path == "" {
		return one, true
	}
	ref, isNote := notes[one.Path]
	if !isNote {
		// The index reported where the link lands but holds no note there: the entry
		// changed during this read.
		return Neighbour{Written: "", Path: "", Title: "", Body: "", Label: "", IsBacklink: false, IsAmbiguous: false, Outcome: ""}, false
	}
	// An ordinary note is what the cards were written from, and is
	// taken below.
	//exhaustive:ignore
	switch kinds[one.Path] {
	case format.TypeCardModel:
		// Every card names the card model it is cut by, so a deck points at
		// its card models. A card model is how a card is laid out and not what
		// it was written from.
		return Neighbour{Written: "", Path: "", Title: "", Body: "", Label: "", IsBacklink: false, IsAmbiguous: false, Outcome: ""}, false
	case format.TypePreset:
		// A deck points at the preset it is scheduled by. A preset is how
		// the cards come round and not what they were written from.
		return Neighbour{Written: "", Path: "", Title: "", Body: "", Label: "", IsBacklink: false, IsAmbiguous: false, Outcome: ""}, false
	case format.TypeDeck:
		// Another deck holds more cards to answer and is not a source of the cards under
		// review.
		return Neighbour{Written: "", Path: "", Title: "", Body: "", Label: "", IsBacklink: false, IsAmbiguous: false, Outcome: ""}, false
	}
	one.Title = ref.Title
	return one, true
}

// readBodies reads the prose of the first MostRead notes that have a path. A
// read fails when the vault itself is out of reach, and a panel of titles with
// no prose under any of them would say nothing about why; what is wrong with
// one note is an outcome, not an error.
func (u ShowNeighbourhood) readBodies(ctx context.Context, v domain.Vault, neighbours []Neighbour) error {
	for i := range neighbours {
		if i == MostRead {
			break
		}
		if neighbours[i].Path == "" {
			continue
		}
		contents, err := u.reads.Execute(ctx, v, neighbours[i].Path)
		if err != nil {
			return err
		}
		neighbours[i].Outcome = contents.Outcome
		neighbours[i].Body = contents.Body
	}
	return nil
}

// AssignCardIDs gives a card id to every card of a vault that carries none.
//
// A card typed manually has no card id until the application writes its file, and
// a card with no card id cannot have an answer recorded against it. A deck
// containing one is written, which creates a card id for every card in it.
//
// This is the only write flashcards makes into a vault. It is made when a session
// starts and not when due cards are counted.
type AssignCardIDs struct {
	readers ports.VaultReaders
	writers ports.VaultWriters
	notes   ports.NoteQueries
	links   ports.LinkQueries
	// index brings what a write touched up to date. A build holding none leaves
	// the index to the next scan.
	index func(ctx context.Context, v domain.Vault, paths []string) error
	// now is when this is happening. A card id written here carries it.
	now ports.Clock
}

// NewAssignCardIDs constructs an AssignCardIDs from the vault readers and writers
// for reading and rewriting decks, the note queries that identify which notes are
// decks, the link queries that resolve the wikilink naming a card's card model,
// the index function that updates the index after a deck is written, and the
// clock.
func NewAssignCardIDs(
	readers ports.VaultReaders,
	writers ports.VaultWriters,
	notes ports.NoteQueries,
	links ports.LinkQueries,
	index func(ctx context.Context, v domain.Vault, paths []string) error,
	now ports.Clock,
) AssignCardIDs {
	return AssignCardIDs{
		readers: readers, writers: writers, notes: notes, links: links,
		index: index, now: now,
	}
}

// WithIndex returns a copy of AssignCardIDs with the given index function.
func (u AssignCardIDs) WithIndex(index func(ctx context.Context, v domain.Vault, paths []string) error) AssignCardIDs {
	u.index = index
	return u
}

// AssignCardIDsResult is what the assigning of card ids came to: the decks it could not write.
type AssignCardIDsResult struct {
	// Unwritten are the paths of the decks holding a card with no card id that
	// could not be given one. Their cards are left out of this session.
	Unwritten []string
}

// Execute writes every deck of the vault that holds a card with no card id.
//
// A deck that could not be written is left as it is and is not an error: the
// editor may be saving it, and the vault's write lock lives in one process. Its
// cards are left out of this session and marked at the next, and it is named in
func (u AssignCardIDs) Execute(ctx context.Context, v domain.Vault) (AssignCardIDsResult, error) {
	if u.notes == nil {
		return AssignCardIDsResult{Unwritten: nil}, nil
	}
	paths, err := u.notes.GetPathsOfType(ctx, v.ID, format.TypeDeck)
	if err != nil {
		return AssignCardIDsResult{}, err
	}

	read := cards.NewRead(u.readers, u.links)
	write := cards.NewWrite(u.readers, u.writers, u.links, u.index, u.now)
	var out AssignCardIDsResult
	for _, path := range paths {
		deck, err := read.Deck(ctx, v, path)
		if err != nil || deck.Outcome != note.Ok || !hasCardWithoutID(deck.Body) {
			continue
		}

		// The body goes back exactly as it was read. What the write is for is
		// the deck being made whole on the way past, which is where a card id is
		// minted.
		body, err := format.DeckBody(deck.Body)
		if err != nil {
			out.Unwritten = append(out.Unwritten, path)
			continue
		}
		// A deck the index could not be brought level with carries its card ids all
		// the same, and its cards stand in this session.
		if _, err := write.Deck(ctx, v, path, body, deck.FileStat); err != nil &&
			!errors.Is(err, note.ErrUnlevelled) {
			out.Unwritten = append(out.Unwritten, path)
		}
	}
	return out, nil
}

// hasCardWithoutID reports whether any card of a deck carries no card id.
func hasCardWithoutID(d format.Deck) bool {
	for _, c := range d.Cards {
		if c.CardID == "" {
			return true
		}
	}
	return false
}
