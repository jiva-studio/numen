// Package service is the backend of the flashcards-studio plugin: the cards, the presets and the
// review that are served to the windows, and the tools it offers an agent. It works the vault
// through the host.
package service

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards/deckedit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/hostports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// The names the schedules and the days counted are kept under in the cache the host holds.
const (
	schedulesCache = "schedules"
	daysCache      = "days"
)

var errNoVault = errors.New("no vault is open")

// Config is what the backend is made from.
type Config struct {
	// Vault is the vault the host lets the plugin work.
	Vault host.Vault
	// Store is the plugin's own folder of the store of each vault.
	Store host.Store
	// DayStart is how long past midnight a day of review begins. Left nil, it is the default.
	DayStart func() time.Duration
	// Now is what time it is. Left nil, it is this machine's clock.
	Now ports.Clock
	// Cores is how many decks are parsed at once. Left zero, one for each CPU.
	Cores int
	// Scheduler is the algorithm that schedules a card. Left nil, FSRS.
	Scheduler ports.ScheduleAlgorithm
}

// Backend answers the cards, the presets and the review.
type Backend struct {
	vault  host.Vault
	ports  hostports.Ports
	titles ports.NoteQueries
	now    ports.Clock
	cfg    Config
	runs   sessions

	// scenariosStore is the scenarios, made again when the settings change.
	scenariosStore atomic.Pointer[Scenarios]
}

// Scenarios is everything the backend does to the cards of a vault, built against the settings in
// force.
type Scenarios struct {
	cards    cards.Scenarios
	deckedit deckedit.Scenarios
	flash    flashcards.Scenarios
	day      review.Day
	// neighbourhood is what a deck is joined to, with the text of each.
	neighbourhood flashcards.ShowNeighbourhood
}

// scenarios is what the backend does to the cards under the settings in force.
func (b *Backend) scenarios() *Scenarios { return b.scenariosStore.Load() }

// New makes the backend over the host's vault.
func New(cfg Config) *Backend {
	wired := hostports.New(cfg.Vault, cfg.Store)
	now := cfg.Now
	if now == nil {
		now = clock.Now
	}
	b := &Backend{vault: cfg.Vault, ports: wired, titles: wired.Notes, now: now, cfg: cfg}
	b.scenariosStore.Store(b.build(cfg.DayStart))
	return b
}

// Reconfigure makes the scenarios again with a day that begins at the hour given. Calls already
// running finish under the settings they began under.
func (b *Backend) Reconfigure(dayStart func() time.Duration) {
	b.scenariosStore.Store(b.build(dayStart))
}

// build makes the scenarios for a day that begins at dayStart.
func (b *Backend) build(dayStart func() time.Duration) *Scenarios {
	cfg, wired, now := b.cfg, b.ports, b.now
	cores := cfg.Cores
	if cores <= 0 {
		cores = runtime.GOMAXPROCS(0)
	}
	// What a window writes is levelled in the index by the paths it touched, and the index is the host's.
	index := func(ctx context.Context, v domain.Vault, paths []string) error {
		return cfg.Vault.With(v.ID).Scan(ctx, paths)
	}

	read := cards.NewRead(wired.Readers, wired.Links)
	write := cards.NewWrite(wired.Readers, wired.Writers, wired.Links, index, now)
	set := cards.Scenarios{
		Read:        read,
		List:        cards.NewList(wired.Readers, wired.Notes),
		Write:       write,
		Create:      cards.NewCreate(wired.Writers, index, now),
		RenameField: cards.NewRenameField(wired.Readers, wired.Writers, wired.Notes, wired.Links, index, now, cards.UnboundedDecks),
		BoundedRenameField: cards.NewRenameField(
			wired.Readers, wired.Writers, wired.Notes, wired.Links, index, now, cards.MaxDecksPerRename),
	}
	deckEdits := deckedit.NewScenarios(read, write)

	running := flashcards.NewScenarios(flashcards.Config{
		Readers:   wired.Readers,
		Writers:   wired.Writers,
		Notes:     wired.Notes,
		Links:     wired.Links,
		Problems:  wired.Problems,
		Index:     index,
		Logs:      wired.Logs,
		Schedules: wired.Schedules(schedulesCache),
		DayCounts: wired.Schedules(daysCache),
		DayStart:  dayStart,
		Now:       now,
		Cores:     cores,
		Scheduler: cfg.Scheduler,
	})
	return &Scenarios{
		cards: set, deckedit: deckEdits, flash: running, day: running.Day,
		neighbourhood: flashcards.NewShowNeighbourhood(note.NewShowLinks(wired.Links), wired.Notes, note.NewRead(wired.Readers)),
	}
}

// getCurrentVault is the vault a question is answered over. A host showing none has no vault to
func (b *Backend) getCurrentVault(ctx context.Context) (domain.Vault, error) {
	v, err := b.vault.GetVault(ctx)
	if errors.Is(err, host.ErrNoVault) {
		return domain.Vault{}, connect.NewError(connect.CodeFailedPrecondition, errNoVault)
	}
	if err != nil {
		return domain.Vault{}, connect.NewError(connect.CodeInternal, err)
	}
	return v, nil
}

// getVault is the vault of an identity, as the host knows it.
func (b *Backend) getVault(ctx context.Context, id string) (domain.Vault, error) {
	v, err := b.vault.With(domain.VaultID(id)).GetVault(ctx)
	if errors.Is(err, host.ErrUnknownVault) {
		return domain.Vault{}, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return domain.Vault{}, connect.NewError(connect.CodeInternal, err)
	}
	return v, nil
}

// getTitle is what the vault calls the note at a path. A path with no note is called nothing.
func (b *Backend) getTitle(ctx context.Context, v domain.Vault, path string) string {
	if path == "" {
		return ""
	}
	found, err := b.titles.GetNotes(ctx, v.ID, []string{path})
	if err != nil {
		return ""
	}
	return found[path].Title
}
