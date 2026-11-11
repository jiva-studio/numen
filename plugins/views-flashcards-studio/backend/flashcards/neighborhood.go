package flashcards

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
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
	Outcome     note.ReadOutcome
}

// neighborNotes is what the neighborhood asks the index of the notes it found.
type neighborNotes interface {
	ports.NoteRefQueries
	ports.NoteTypeQueries
}

// ShowNeighborhood is what the deck being reviewed is joined to, with the text
// of each.
//
// A card's links are written in the deck's file, so a deck already points at
// everything its cards name: this is the reading beside the review.
type ShowNeighborhood struct {
	linked note.ShowLinks
	notes  neighborNotes
	reads  note.Read
}

// NewShowNeighborhood constructs a ShowNeighborhood from the source of the
// deck's links, the note queries that identify which of those notes are decks and
// card models, and the reader of each note's prose.
//
// The prose is read only where a deck names something.
func NewShowNeighborhood(
	linked note.ShowLinks, notes neighborNotes, reads note.Read,
) ShowNeighborhood {
	return ShowNeighborhood{linked: linked, notes: notes, reads: reads}
}

// Execute gathers what the deck is joined to: what it points at first, then what points at it.
// An error is the vault or the index being out of reach. What is wrong with one note (gone, too
// long, not a note) is an outcome on that note's own entry, and the rest are still read.
func (u ShowNeighborhood) Execute(
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

func getNeighbors(v domain.Vault, deck string, linked note.Links) []Neighbor {
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

func (u ShowNeighborhood) readBodies(ctx context.Context, v domain.Vault, neighbors []Neighbor) error {
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

// AssignCardIDs gives a card id to every card of a vault that carries none.
// A card typed manually has no card id until the application writes its file, and a card without
// one cannot have an answer recorded against it. This is the only write flashcards makes into a
// vault, and it is made when a session starts, not when due cards are counted.
type AssignCardIDs struct {
	readers ports.VaultReaders
	writers ports.VaultWriters
	notes   ports.TypePathQueries
	links   ports.LinkQueries
	index   func(ctx context.Context, v domain.Vault, paths []string) error
	now     ports.Clock
}

// NewAssignCardIDs constructs an AssignCardIDs from the vault readers and writers
// for reading and rewriting decks, the note queries that identify which notes are
// decks, the link queries that resolve the wikilink naming a card's card model,
// the index function that updates the index after a deck is written, and the
// clock.
func NewAssignCardIDs(
	readers ports.VaultReaders,
	writers ports.VaultWriters,
	notes ports.TypePathQueries,
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
		// gomutants:disable-next-line BRANCH_IF reason="Parsed decks always have valid section indices and card model links."
		if err != nil {
			out.Unwritten = append(out.Unwritten, path)
			continue
		}
		// A deck the index could not be updated for carries its card ids all
		// the same, and its cards stand in this session.
		if _, err := write.Deck(ctx, v, path, body, deck.FileStat); err != nil &&
			!errors.Is(err, note.ErrIndexOutdated) {
			out.Unwritten = append(out.Unwritten, path)
		}
	}
	return out, nil
}

func hasCardWithoutID(d format.Deck) bool {
	for _, c := range d.Cards {
		if c.CardID == "" {
			return true
		}
	}
	return false
}
