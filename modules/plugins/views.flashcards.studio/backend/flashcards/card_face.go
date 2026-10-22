package flashcards

import (
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/cards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/note"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
)

// CardFace is one card face as the vault holds it: a card, shown through one
// face of the stencil that cuts it, and where both of them are written.
type CardFace struct {
	// the section it is placed under. A card standing before the first section has
	// none.
	Deck    string
	Section string
	// ID is what a schedule belongs to: the card's mark and the face's name.
	ID review.CardFaceID
	// Heading is what the card's heading shows, which is the first line of its
	// first field.
	Heading string

	// What it is laid out from. It is read once with the deck and kept,
	// so laying out is the last thing done and only for a card about to be
	// shown: counting what a vault owes fills in no template at all.
	stencil format.Stencil
	face    format.FaceTemplate
	card    format.Card
}

// Lay is the face filled with this card's values: what stands before the answer
// and what stands after it, as HTML.
func (s CardFace) Lay() (front, back string) { return format.Lay(s.face, s.card) }

// ListCardFaces is every card face a vault holds, read out of its files.
//
// Nothing here writes. A card carrying no mark cannot be answered and is left
// out; giving it one is Marking, which a person's own vault has done to it
// before they are asked anything.
type ListCardFaces struct {
	readers ports.VaultReaders
	// notes says which notes of the vault are decks. A build holding none
	// answers ErrNotCarried for every vault, the same as an index that has not
	// read this one yet.
	notes ports.NoteQueries
	links ports.LinkQueries
	// cache is the faces of the decks already parsed, for the life of the window.
	// Copies of a ListCardFaces share it.
	cache *faceCache
	// Cores is how many decks are parsed at once. A build naming none parses one
	// at a time.
	Cores int
	// MaxFaces is the most card faces kept parsed. A build naming none keeps
	// defaultCachedFaces.
	MaxFaces int
}

// NewListCardFaces is what a vault's card faces are read through: the vault the
// decks and the stencils are read out of, what says which of its notes are
// decks, and where the wikilink a card names its stencil by lands.
func NewListCardFaces(
	readers ports.VaultReaders, notes ports.NoteQueries, links ports.LinkQueries,
) ListCardFaces {
	return ListCardFaces{readers: readers, notes: notes, links: links, cache: newFaceCache()}
}

// WithReaders returns a copy of ListCardFaces with the given VaultReaders.
func (u ListCardFaces) WithReaders(readers ports.VaultReaders) ListCardFaces {
	u.readers = readers
	u.cache = newFaceCache()
	return u
}

// ErrNotCarried is what a vault the index does not carry gets. It is the signal
// to read that vault, and a window holding one reads it.
var ErrNotCarried = errors.New("the index does not carry this vault yet")

// the index does not carry gets ErrNotCarried.
func (u ListCardFaces) Execute(ctx context.Context, v domain.Vault) ([]CardFace, error) {
	paths, err := u.GetDeckPaths(ctx, v)
	if err != nil {
		return nil, err
	}
	faces := u.GetFaces(ctx, v, paths)
	u.cache.keep(v.ID, paths)
	return faces, nil
}

// GetDeckPaths returns the path of every deck the vault holds.
//
// A vault the index does not carry gets ErrNotCarried. The list of decks is the
// index's answer, and a vault absent from it is not a vault holding no cards.
func (u ListCardFaces) GetDeckPaths(ctx context.Context, v domain.Vault) ([]string, error) {
	if u.notes == nil {
		return nil, ErrNotCarried
	}
	target, err := u.notes.Has(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	if !target {
		return nil, ErrNotCarried
	}
	return u.notes.GetPathsOfType(ctx, v.ID, format.TypeDeck)
}

// cards of one preset hands it the decks pointing there, and the rest of the
// vault is left unread.
//
// A deck whose file and stencils are the ones its faces were last parsed from is
// not parsed again. The decks that are parsed are parsed on up to Cores places at
// once, and the faces come back in the order of paths.
//
// A deck that cannot be read contributes no card face and is not an error. What
// was wrong with it is the deck's own problem, and the editor is where it is
// settled.
func (u ListCardFaces) GetFaces(ctx context.Context, v domain.Vault, paths []string) []CardFace {
	epoch := u.cache.begin()
	reader, err := u.readers.Open(v)
	found := make([][]CardFace, len(paths))
	var misses []int
	for item, path := range paths {
		if err == nil {
			if faces, target := u.getCached(ctx, reader, v, path); target {
				found[item] = faces
				continue
			}
		}
		misses = append(misses, item)
	}

	stencils := &stencilMemo{read: cards.NewRead(u.readers, u.links), stencils: make(map[string]stencilRead)}
	places := min(max(1, u.Cores), len(misses))
	var running sync.WaitGroup
	queue := make(chan int)
	for range places {
		running.Add(1)
		go func() {
			defer running.Done()
			for entry := range queue {
				found[entry] = u.parseDeck(ctx, v, paths[entry], epoch, stencils)
			}
		}()
	}
	for _, elem := range misses {
		queue <- elem
	}
	close(queue)
	running.Wait()

	var out []CardFace
	for _, faces := range found {
		out = append(out, faces...)
	}
	return out
}

// Invalidate forgets the decks at paths and every deck cut by a stencil at one
// of them.
func (u ListCardFaces) Invalidate(vault domain.VaultID, paths ...string) {
	u.cache.invalidate(vault, paths)
}

// InvalidateAll forgets every deck parsed.
func (u ListCardFaces) InvalidateAll() { u.cache.invalidateAll() }

// getCached is the faces of the deck at path when the files they were parsed
// from are the files now standing.
func (u ListCardFaces) getCached(
	ctx context.Context, reader ports.VaultReader, v domain.Vault, path string,
) ([]CardFace, bool) {
	key := deckKey{vault: v.ID, path: path}
	e, target := u.cache.get(key)
	if !target {
		return nil, false
	}
	deck, err := reader.GetFingerprint(ctx, path)
	if err != nil {
		u.cache.drop(key)
		return nil, false
	}
	stencils := make(map[string]domain.Fingerprint, len(e.stencils))
	for stencil := range e.stencils {
		pos, err := reader.GetFingerprint(ctx, stencil)
		if err != nil {
			u.cache.drop(key)
			return nil, false
		}
		stencils[stencil] = pos
	}
	if !e.isCurrent(deck, stencils) {
		u.cache.drop(key)
		return nil, false
	}
	return slices.Clone(e.faces), true
}

// parseDeck reads one deck and lays out its faces. They are stored when every
// stencil a card named was read.
func (u ListCardFaces) parseDeck(
	ctx context.Context, v domain.Vault, path string, epoch uint64, stencils *stencilMemo,
) []CardFace {
	deck, err := stencils.read.Deck(ctx, v, path)
	if err != nil || deck.Outcome != note.Ok {
		return nil
	}
	faces, used, isWhole := u.getDeckFaces(ctx, v, deck, stencils)
	if isWhole {
		u.cache.put(epoch, deckKey{vault: v.ID, path: path}, deckFaces{
			deck: deck.Fingerprint, stencils: used, faces: slices.Clone(faces),
		}, u.getFaceLimit())
	}
	return faces
}

// stencilMemo is the stencils one request has read, shared by the places
// parsing it.
type stencilMemo struct {
	read cards.Read
	mu   sync.Mutex
	stencils map[string]stencilRead
}

// stencilRead is a stencil as the request read it. A note that is not a stencil
// carries a stencil of nothing and is not whole.
type stencilRead struct {
	stencil     format.Stencil
	fingerprint domain.Fingerprint
	isWhole     bool
}

func (m *stencilMemo) get(ctx context.Context, v domain.Vault, path string) stencilRead {
	m.mu.Lock()
	defer m.mu.Unlock()
	if target, observed := m.stencils[path]; observed {
		return target
	}
	// A card whose stencil is not one is a card no face shows. It is a problem
	// against the deck, and it is settled in a window.
	out := stencilRead{}
	one, err := m.read.Stencil(ctx, v, path)
	if err == nil && one.Outcome == note.Ok && one.Type == format.TypeStencil {
		out = stencilRead{stencil: one.Body, fingerprint: one.Fingerprint, isWhole: true}
	}
	m.stencils[path] = out
	return out
}

// getDeckFaces returns the card faces one deck holds: every card of a mark, through every
// face of the stencil it names that lays anything out. It also says which
// stencils the faces stand on, and whether every stencil a card named was read.
func (u ListCardFaces) getDeckFaces(
	ctx context.Context, v domain.Vault, deck cards.DeckContents, stencils *stencilMemo,
) (out []CardFace, used map[string]domain.Fingerprint, isWhole bool) {
	used = make(map[string]domain.Fingerprint)
	isWhole = true
	for _, card := range deck.Body.Cards {
		if card.Mark == "" {
			continue
		}
		path, target := deck.Stencils[card.StencilLink]
		if !target {
			continue
		}
		one := stencils.get(ctx, v, path)
		if !one.isWhole {
			isWhole = false
			continue
		}
		used[path] = one.fingerprint

		for _, face := range one.stencil.Faces {
			// A face missing a side lays out nothing, which is a problem
			// against the stencil and no card face here.
			if face.Front == "" || face.Back == "" {
				continue
			}
			out = append(out, CardFace{
				Deck:    deck.Path,
				Section: getSectionName(deck.Body, card),
				ID:      review.CardFaceID{Card: string(card.Mark), Face: face.Name},
				Heading: card.Heading,
				stencil: one.stencil,
				face:    face,
				card:    card,
			})
		}
	}
	return out, used, isWhole
}

// getSectionName returns the name of the section a card is placed under, and is empty for a
// card standing before the first.
func getSectionName(d format.Deck, c format.Card) string {
	if c.Section == format.NoSection || c.Section >= len(d.Sections) {
		return ""
	}
	return d.Sections[c.Section].Name
}

func (u ListCardFaces) getFaceLimit() int {
	if u.MaxFaces > 0 {
		return u.MaxFaces
	}
	return defaultCachedFaces
}
