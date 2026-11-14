package flashcards

import (
	"container/list"
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// ReviewCard is one review card as the vault holds it: a card, shown through one
// review card of the card model that cuts it, and where both of them are written.
type ReviewCard struct {
	Deck    string
	Section string
	ID      review.CardID
	Heading string

	cardModel  format.CardModel
	reviewCard format.CardTemplate
	card       format.Card
}

// Lay returns the review card filled with this card's values: the front shown
// before the answer and the back shown after it, as HTML.
func (s ReviewCard) Lay() (front, back string) { return format.Lay(s.reviewCard, s.card) }

// reviewCardNotes is what listing review cards asks the index: whether it holds the vault, and
// which notes are decks or presets.
type reviewCardNotes interface {
	ports.IndexQueries
	ports.TypePathQueries
}

// ListReviewCards returns every review card a vault holds, read from its files.
//
// It writes nothing. A card without a card id cannot be answered and is
// excluded; assigning card ids is done when the vault's decks are written, before
// any review card is requested.
type ListReviewCards struct {
	readers  ports.VaultReaders
	notes    reviewCardNotes
	read     ports.CardNoteReader
	cache    *reviewCardCache
	Cores    int
	MaxCards int
}

// NewListReviewCards returns a ListReviewCards that reads review cards through
// the vault holding the decks and the card models, the query that identifies
// which notes are decks, and the reader of a deck and of a card model.
func NewListReviewCards(
	readers ports.VaultReaders, notes reviewCardNotes, read ports.CardNoteReader,
) ListReviewCards {
	return ListReviewCards{readers: readers, notes: notes, read: read, cache: newCardCache(), Cores: 0, MaxCards: 0}
}

// ErrNotCarried is what a vault the index does not carry gets. It is the signal
// to read that vault, and a window holding one reads it.
var ErrNotCarried = errors.New("the index does not contain this vault yet")

// Execute lists the review cards of a vault. A vault the index does not carry gets ErrNotCarried.
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

// GetReviewCards returns the review cards of the decks at paths, in the order of paths. A
// caller needing one preset's cards passes the decks pointing at it, and the rest of the vault is
// not read. A deck whose file and card models match its last parse is not parsed again; the
// others are parsed concurrently on up to Cores workers. A deck that cannot be read contributes
// no card and is not an error, since the editor reports it as a problem of the deck.
func (u ListReviewCards) GetReviewCards(ctx context.Context, v domain.Vault, paths []string) []ReviewCard {
	epoch := u.cache.begin()
	reader, err := u.readers.Open(v)
	found := make([][]ReviewCard, len(paths))
	var misses []int
	for item, path := range paths {
		if err == nil {
			if reviewCards, target := u.getCachedCards(ctx, reader, v, path); target {
				found[item] = reviewCards
				continue
			}
		}
		misses = append(misses, item)
	}

	cardModels := &cardModelMemo{read: u.read, mu: sync.Mutex{}, cardModels: make(map[string]cardModelRead)}
	// gomutants:disable-next-line INTEGER_INCREMENT reason="Minimum worker count affects concurrency bound only."
	places := min(max(1, u.Cores), len(misses))
	var running sync.WaitGroup
	queue := make(chan int)
	for range places {
		running.Go(func() {
			for entry := range queue {
				found[entry] = u.parseDeck(ctx, v, paths[entry], epoch, cardModels)
			}
		})
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

func (u ListReviewCards) getCachedCards(
	ctx context.Context, reader ports.VaultReader, v domain.Vault, path string,
) ([]ReviewCard, bool) {
	key := deckKey{vault: v.ID, path: path}
	e, target := u.cache.get(key)
	if !target {
		return nil, false
	}
	deck, err := reader.StatFile(ctx, path)
	if err != nil {
		u.cache.drop(key)
		return nil, false
	}
	cardModels := make(map[string]domain.FileStat, len(e.cardModels))
	for cardModel := range e.cardModels {
		pos, err := reader.StatFile(ctx, cardModel)
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

func (u ListReviewCards) parseDeck(
	ctx context.Context, v domain.Vault, path string, epoch uint64, cardModels *cardModelMemo,
) []ReviewCard {
	deck, err := cardModels.read.Deck(ctx, v, path)
	if err != nil || deck.Outcome != edit.Ok {
		return nil
	}
	reviewCards, used, isWhole := u.getDeckCards(ctx, v, deck, cardModels)
	if isWhole {
		u.cache.put(epoch, deckKey{vault: v.ID, path: path}, deckCards{
			deck: deck.FileStat, cardModels: used, reviewCards: slices.Clone(reviewCards),
		}, u.getCardLimit())
	}
	return reviewCards
}

// cardModelMemo is the card models one request has read, shared by the places
// parsing it.
type cardModelMemo struct {
	read       ports.CardNoteReader
	mu         sync.Mutex
	cardModels map[string]cardModelRead
}

// cardModelRead is a card model as the request read it. A note that is not a card model
// carries a card model of nothing and is not whole.
type cardModelRead struct {
	cardModel format.CardModel
	stat      domain.FileStat
	isWhole   bool
}

func (m *cardModelMemo) get(ctx context.Context, v domain.Vault, path string) cardModelRead {
	m.mu.Lock()
	defer m.mu.Unlock()
	if target, observed := m.cardModels[path]; observed {
		return target
	}
	// A card whose card model is not one is a card no review card shows. It is a problem
	// against the deck, and it is settled in a window.
	out := cardModelRead{
		cardModel: format.CardModel{Ref: domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}, Fields: nil, Preamble: "", Tail: "", Templates: nil, Problems: nil},
		stat:      domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}, isWhole: false,
	}
	one, err := m.read.CardModel(ctx, v, path)
	if err == nil && one.Outcome == edit.Ok && one.Type == format.TypeCardModel {
		out = cardModelRead{cardModel: one.Body, stat: one.FileStat, isWhole: true}
	}
	m.cardModels[path] = out
	return out
}

func (u ListReviewCards) getDeckCards(
	ctx context.Context, v domain.Vault, deck format.DeckContents, cardModels *cardModelMemo,
) (out []ReviewCard, used map[string]domain.FileStat, isWhole bool) {
	used = make(map[string]domain.FileStat)
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
		used[path] = one.stat

		for _, reviewCard := range one.cardModel.Templates {
			// A review card missing a side lays out nothing, which is a problem
			// against the card model and no review card here.
			if reviewCard.Front == "" || reviewCard.Back == "" {
				continue
			}
			out = append(out, ReviewCard{
				Deck:       deck.Path,
				Section:    getSectionName(deck.Body, card),
				ID:         review.CardID{Card: string(card.CardID), Template: reviewCard.Name},
				Heading:    card.Heading,
				cardModel:  one.cardModel,
				reviewCard: reviewCard,
				card:       card,
			})
		}
	}
	return out, used, isWhole
}

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
	deck        domain.FileStat
	cardModels  map[string]domain.FileStat
	reviewCards []ReviewCard
}

func (e deckCards) isCurrent(deck domain.FileStat, cardModels map[string]domain.FileStat) bool {
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
	order *list.List
	total int
	epoch uint64
}

func newCardCache() *reviewCardCache {
	return &reviewCardCache{mu: sync.Mutex{}, decks: make(map[deckKey]*list.Element), order: list.New(), total: 0, epoch: 0}
}

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
		// gomutants:disable-next-line RETURN_ZERO reason="Struct literal already holds the zero deckCards value."
		return deckCards{deck: domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}, cardModels: nil, reviewCards: nil}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	pos, target := c.decks[key]
	if !target {
		// gomutants:disable-next-line RETURN_ZERO reason="Struct literal already holds the zero deckCards value."
		return deckCards{deck: domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}, cardModels: nil, reviewCards: nil}, false
	}
	c.order.MoveToFront(pos)
	return getEntry(pos).deckCards, true
}

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

func usesAny(cardModels map[string]domain.FileStat, paths []string) bool {
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

func getEntry(pos *list.Element) *cached {
	one, isCached := pos.Value.(*cached)
	if !isCached || one == nil {
		return &cached{key: deckKey{vault: "", path: ""}, deckCards: deckCards{deck: domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}, cardModels: nil, reviewCards: nil}}
	}
	return one
}
