package flashcards

import (
	"container/list"
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// ReviewCard is one review card as the vault holds it: a card, shown through one
// review card of the card model that cuts it, and where both of them are written.
type ReviewCard struct {
	// the section it is placed under. A card standing before the first section has
	// none.
	Deck    string
	Section string
	// ID is what a schedule belongs to: the card's id and the template's name.
	ID review.ReviewCardID
	// Heading is what the card's heading shows, which is the first line of its
	// first field.
	Heading string

	// What it is laid out from. It is read once with the deck and kept,
	// so laying out is the last thing done and only for a card about to be
	// shown: counting what a vault owes fills in no template at all.
	cardModel  format.CardModel
	reviewCard format.CardTemplate
	card       format.Card
}

// Lay is the review card filled with this card's values: what stands before the answer
// and what stands after it, as HTML.
func (s ReviewCard) Lay() (front, back string) { return format.Lay(s.reviewCard, s.card) }

// ListReviewCards is every review card a vault holds, read out of its files.
//
// Nothing here writes. A card carrying no card id cannot be answered and is left
// out; giving it one is assigning card ids, which a person's own vault has done to it
// before they are asked anything.
type ListReviewCards struct {
	readers ports.VaultReaders
	// notes says which notes of the vault are decks. A build holding none
	// answers ErrNotCarried for every vault, the same as an index that has not
	// read this one yet.
	notes ports.NoteQueries
	links ports.LinkQueries
	// cache is the review cards of the decks already parsed, for the life of the window.
	// Copies of a ListReviewCards share it.
	cache *reviewCardCache
	// Cores is how many decks are parsed at once. A build naming none parses one
	// at a time.
	Cores int
	// MaxCards is the most review cards kept parsed. A build naming none keeps
	// defaultCachedCards.
	MaxCards int
}

// NewListReviewCards is what a vault's review cards are read through: the vault the
// decks and the card models are read out of, what says which of its notes are
// decks, and where the wikilink a card names its card model by lands.
func NewListReviewCards(
	readers ports.VaultReaders, notes ports.NoteQueries, links ports.LinkQueries,
) ListReviewCards {
	return ListReviewCards{readers: readers, notes: notes, links: links, cache: newCardCache()}
}

// WithReaders returns a copy of ListReviewCards with the given VaultReaders.
func (u ListReviewCards) WithReaders(readers ports.VaultReaders) ListReviewCards {
	u.readers = readers
	u.cache = newCardCache()
	return u
}

// ErrNotCarried is what a vault the index does not carry gets. It is the signal
// to read that vault, and a window holding one reads it.
var ErrNotCarried = errors.New("the index does not carry this vault yet")

// the index does not carry gets ErrNotCarried.
func (u ListReviewCards) Execute(ctx context.Context, v domain.Vault) ([]ReviewCard, error) {
	paths, err := u.GetDeckPaths(ctx, v)
	if err != nil {
		return nil, err
	}
	reviewCards := u.GetReviewCards(ctx, v, paths)
	u.cache.keep(v.ID, paths)
	return reviewCards, nil
}

// GetDeckPaths returns the path of every deck the vault holds.
//
// A vault the index does not carry gets ErrNotCarried. The list of decks is the
// index's answer, and a vault absent from it is not a vault holding no cards.
func (u ListReviewCards) GetDeckPaths(ctx context.Context, v domain.Vault) ([]string, error) {
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
// A deck whose file and card models are the ones its review cards were last parsed from is
// not parsed again. The decks that are parsed are parsed on up to Cores places at
// once, and the review cards come back in the order of paths.
//
// A deck that cannot be read contributes no review card and is not an error. What
// was wrong with it is the deck's own problem, and the editor is where it is
// settled.
func (u ListReviewCards) GetReviewCards(ctx context.Context, v domain.Vault, paths []string) []ReviewCard {
	epoch := u.cache.begin()
	reader, err := u.readers.Open(v)
	found := make([][]ReviewCard, len(paths))
	var misses []int
	for item, path := range paths {
		if err == nil {
			if reviewCards, target := u.getCached(ctx, reader, v, path); target {
				found[item] = reviewCards
				continue
			}
		}
		misses = append(misses, item)
	}

	cardModels := &cardModelMemo{read: cards.NewRead(u.readers, u.links), cardModels: make(map[string]cardModelRead)}
	places := min(max(1, u.Cores), len(misses))
	var running sync.WaitGroup
	queue := make(chan int)
	for range places {
		running.Add(1)
		go func() {
			defer running.Done()
			for entry := range queue {
				found[entry] = u.parseDeck(ctx, v, paths[entry], epoch, cardModels)
			}
		}()
	}
	for _, elem := range misses {
		queue <- elem
	}
	close(queue)
	running.Wait()

	var out []ReviewCard
	for _, reviewCards := range found {
		out = append(out, reviewCards...)
	}
	return out
}

// Invalidate forgets the decks at paths and every deck cut by a card model at one
// of them.
func (u ListReviewCards) Invalidate(vault domain.VaultID, paths ...string) {
	u.cache.invalidate(vault, paths)
}

// InvalidateAll forgets every deck parsed.
func (u ListReviewCards) InvalidateAll() { u.cache.invalidateAll() }

// getCached is the review cards of the deck at path when the files they were parsed
// from are the files now standing.
func (u ListReviewCards) getCached(
	ctx context.Context, reader ports.VaultReader, v domain.Vault, path string,
) ([]ReviewCard, bool) {
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
	cardModels := make(map[string]domain.Fingerprint, len(e.cardModels))
	for cardModel := range e.cardModels {
		pos, err := reader.GetFingerprint(ctx, cardModel)
		if err != nil {
			u.cache.drop(key)
			return nil, false
		}
		cardModels[cardModel] = pos
	}
	if !e.isCurrent(deck, cardModels) {
		u.cache.drop(key)
		return nil, false
	}
	return slices.Clone(e.reviewCards), true
}

// parseDeck reads one deck and lays out its review cards. They are stored when every
// card model a card named was read.
func (u ListReviewCards) parseDeck(
	ctx context.Context, v domain.Vault, path string, epoch uint64, cardModels *cardModelMemo,
) []ReviewCard {
	deck, err := cardModels.read.Deck(ctx, v, path)
	if err != nil || deck.Outcome != note.Ok {
		return nil
	}
	reviewCards, used, isWhole := u.getDeckCards(ctx, v, deck, cardModels)
	if isWhole {
		u.cache.put(epoch, deckKey{vault: v.ID, path: path}, deckCards{
			deck: deck.Fingerprint, cardModels: used, reviewCards: slices.Clone(reviewCards),
		}, u.getCardLimit())
	}
	return reviewCards
}

// cardModelMemo is the card models one request has read, shared by the places
// parsing it.
type cardModelMemo struct {
	read       cards.Read
	mu         sync.Mutex
	cardModels map[string]cardModelRead
}

// cardModelRead is a card model as the request read it. A note that is not a card model
// carries a card model of nothing and is not whole.
type cardModelRead struct {
	cardModel   format.CardModel
	fingerprint domain.Fingerprint
	isWhole     bool
}

func (m *cardModelMemo) get(ctx context.Context, v domain.Vault, path string) cardModelRead {
	m.mu.Lock()
	defer m.mu.Unlock()
	if target, observed := m.cardModels[path]; observed {
		return target
	}
	// A card whose card model is not one is a card no review card shows. It is a problem
	// against the deck, and it is settled in a window.
	out := cardModelRead{}
	one, err := m.read.CardModel(ctx, v, path)
	if err == nil && one.Outcome == note.Ok && one.Type == format.TypeCardModel {
		out = cardModelRead{cardModel: one.Body, fingerprint: one.Fingerprint, isWhole: true}
	}
	m.cardModels[path] = out
	return out
}

// getDeckCards returns the review cards one deck holds: every card of a card id, through every
// review card of the card model it names that lays anything out. It also says which
// card models the review cards stand on, and whether every card model a card named was read.
func (u ListReviewCards) getDeckCards(
	ctx context.Context, v domain.Vault, deck cards.DeckContents, cardModels *cardModelMemo,
) (out []ReviewCard, used map[string]domain.Fingerprint, isWhole bool) {
	used = make(map[string]domain.Fingerprint)
	isWhole = true
	for _, card := range deck.Body.Cards {
		if card.CardID == "" {
			continue
		}
		path, target := deck.CardModels[card.CardModelLink]
		if !target {
			continue
		}
		one := cardModels.get(ctx, v, path)
		if !one.isWhole {
			isWhole = false
			continue
		}
		used[path] = one.fingerprint

		for _, reviewCard := range one.cardModel.Templates {
			// A review card missing a side lays out nothing, which is a problem
			// against the card model and no review card here.
			if reviewCard.Front == "" || reviewCard.Back == "" {
				continue
			}
			out = append(out, ReviewCard{
				Deck:       deck.Path,
				Section:    getSectionName(deck.Body, card),
				ID:         review.ReviewCardID{Card: string(card.CardID), Template: reviewCard.Name},
				Heading:    card.Heading,
				cardModel:  one.cardModel,
				reviewCard: reviewCard,
				card:       card,
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

func (u ListReviewCards) getCardLimit() int {
	if u.MaxCards > 0 {
		return u.MaxCards
	}
	return defaultCachedCards
}

// defaultCachedCards is how many review cards a window keeps parsed when
// ListReviewCards.MaxCards names none. Past it the least recently used decks are
// forgotten one at a time.
const defaultCachedCards = 1 << 18

type deckKey struct {
	vault domain.VaultID
	path  string
}

// deckCards is the review cards one deck held when it was parsed, and the files
// they were laid out from. It is read-only once stored.
type deckCards struct {
	deck        domain.Fingerprint
	cardModels  map[string]domain.Fingerprint
	reviewCards []ReviewCard
}

// isCurrent says whether the deck and every card model it was cut by are the files
// the review cards were parsed from.
func (e deckCards) isCurrent(deck domain.Fingerprint, cardModels map[string]domain.Fingerprint) bool {
	if !e.deck.IsUnchanged(deck) || len(e.cardModels) != len(cardModels) {
		return false
	}
	for path, one := range e.cardModels {
		other, target := cardModels[path]
		if !target || !one.IsUnchanged(other) {
			return false
		}
	}
	return true
}

type cached struct {
	key deckKey
	deckCards
}

// reviewCardCache is the parsed review cards of the decks a window has read, for the life of
// the window. A nil cache holds nothing.
type reviewCardCache struct {
	mu    sync.Mutex
	decks map[deckKey]*list.Element
	// order runs from the most recently used deck to the least.
	order *list.List
	// total is the number of review cards held.
	total int
	// epoch counts invalidations, and a parse that began before one is not
	// stored.
	epoch uint64
}

func newCardCache() *reviewCardCache {
	return &reviewCardCache{decks: make(map[deckKey]*list.Element), order: list.New()}
}

// begin returns the epoch a parse starts in.
func (c *reviewCardCache) begin() uint64 {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.epoch
}

func (c *reviewCardCache) get(key deckKey) (deckCards, bool) {
	if c == nil {
		return deckCards{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	pos, target := c.decks[key]
	if !target {
		return deckCards{}, false
	}
	c.order.MoveToFront(pos)
	return getEntry(pos).deckCards, true
}

// put stores a deck parsed in the given epoch, then forgets the least recently
// used decks until no more than limit review cards are held. A deck holding more than
// limit review cards is not stored.
func (c *reviewCardCache) put(epoch uint64, key deckKey, e deckCards, limit int) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if epoch != c.epoch || len(e.reviewCards) > limit {
		return
	}
	c.remove(key)
	c.decks[key] = c.order.PushFront(&cached{key: key, deckCards: e})
	c.total += len(e.reviewCards)
	for c.total > limit {
		c.removeElement(c.order.Back())
	}
}

// remove forgets the deck at key. The caller holds the lock.
func (c *reviewCardCache) remove(key deckKey) {
	if pos, target := c.decks[key]; target {
		c.removeElement(pos)
	}
}

func (c *reviewCardCache) removeElement(pos *list.Element) {
	one := getEntry(pos)
	c.order.Remove(pos)
	delete(c.decks, one.key)
	c.total -= len(one.reviewCards)
}

func (c *reviewCardCache) drop(key deckKey) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.remove(key)
}

// keep drops every entry of the vault whose path is not in paths.
func (c *reviewCardCache) keep(vault domain.VaultID, paths []string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.decks {
		if key.vault == vault && !slices.Contains(paths, key.path) {
			c.remove(key)
		}
	}
}

// invalidate drops the decks at paths and every deck cut by a card model at one of
// them.
func (c *reviewCardCache) invalidate(vault domain.VaultID, paths []string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.epoch++
	for key, item := range c.decks {
		if key.vault != vault {
			continue
		}
		if slices.Contains(paths, key.path) || usesAny(getEntry(item).cardModels, paths) {
			c.removeElement(item)
		}
	}
}

func usesAny(cardModels map[string]domain.Fingerprint, paths []string) bool {
	for _, path := range paths {
		if _, target := cardModels[path]; target {
			return true
		}
	}
	return false
}

func (c *reviewCardCache) invalidateAll() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.epoch++
	clear(c.decks)
	c.order.Init()
	c.total = 0
}

// getSize returns the number of review cards held.
func (c *reviewCardCache) getSize() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.total
}

// hasDeck says whether the deck at key is held.
func (c *reviewCardCache) hasDeck(key deckKey) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, target := c.decks[key]
	return target
}

// getEntry returns the deck an element of the order stands for. Only decks are put in
// it, so the zero entry is never seen.
func getEntry(pos *list.Element) *cached {
	one, _ := pos.Value.(*cached)
	if one == nil {
		return &cached{}
	}
	return one
}
