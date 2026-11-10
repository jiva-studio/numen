// Package service is the backend of the flashcards-studio plugin: the cards, the presets and the
// review that are served to the windows, and the tools it offers an agent. It works the vault
// through the host.
package service

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards/deckedit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/hostports"
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
	Vaults    func(domain.VaultID) hostports.Vault
	Store     func(domain.VaultID) hostports.Folder
	DayStart  func() time.Duration
	Now       ports.Clock
	Cores     int
	Scheduler ports.ScheduleAlgorithm
}

// Backend answers the cards, the presets and the review.
type Backend struct {
	vaults func(domain.VaultID) hostports.Vault
	ports  hostports.Ports
	titles ports.NoteQueries
	now    ports.Clock
	cfg    Config
	runs   sessions

	scenariosStore atomic.Pointer[Scenarios]
}

// Scenarios is everything the backend does to the cards of a vault, built against the settings in
// force.
type Scenarios struct {
	cards        cards.Scenarios
	deckedit     deckedit.Scenarios
	flash        flashcards.Scenarios
	day          review.Day
	neighborhood flashcards.ShowNeighbourhood
}

func (b *Backend) scenarios() *Scenarios { return b.scenariosStore.Load() }

// New makes the backend over the host's vault.
func New(cfg Config) *Backend {
	wired := hostports.New(cfg.Vaults, cfg.Store)
	now := cfg.Now
	b := &Backend{
		vaults: cfg.Vaults, ports: wired, titles: wired.Notes, now: now, cfg: cfg,
		runs: sessions{mu: sync.Mutex{}, run: nil}, scenariosStore: atomic.Pointer[Scenarios]{},
	}
	b.scenariosStore.Store(b.build(cfg.DayStart))
	return b
}

// Reconfigure makes the scenarios again with a day that begins at the hour given. Calls already
// running finish under the settings they began under.
func (b *Backend) Reconfigure(dayStart func() time.Duration) {
	b.scenariosStore.Store(b.build(dayStart))
}

func (b *Backend) build(dayStart func() time.Duration) *Scenarios {
	cfg, wired, now := b.cfg, b.ports, b.now
	cores := cfg.Cores
	if cores <= 0 {
		// gomutants:disable-next-line INTEGER_DECREMENT reason="Any non-positive argument queries GOMAXPROCS without changing it."
		cores = runtime.GOMAXPROCS(0)
	}
	// What a window writes is updated in the index by the paths it touched, and the index is the host's.
	index := func(ctx context.Context, v domain.Vault, paths []string) error {
		return cfg.Vaults(v.ID).Index.Scan(ctx, paths)
	}

	read := cards.NewRead(wired.Readers, wired.Links)
	write := cards.NewWrite(wired.Readers, wired.Writers, wired.Links, index, now)
	set := cards.Scenarios{
		Read:        read,
		List:        cards.NewList(wired.Readers, wired.Notes),
		Write:       write,
		Create:      cards.NewCreate(wired.Writers, index, now),
		RenameField: cards.NewRenameField(wired.Readers, wired.Writers, wired.Notes, wired.Links, index, now),
		BoundedRenameField: cards.NewRenameField(
			wired.Readers, wired.Writers, wired.Notes, wired.Links, index, now).WithMaxDecks(cards.MaxDecksPerRename),
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
		neighborhood: flashcards.NewShowNeighbourhood(note.NewShowLinks(wired.Links), wired.Notes, note.NewRead(wired.Readers)),
	}
}

func (b *Backend) getCurrentVault(ctx context.Context) (domain.Vault, error) {
	v, err := b.vaults("").Selector.GetVault(ctx)
	if errors.Is(err, host.ErrNoVault) {
		return domain.Vault{}, newError(server.CodeFailedPrecondition, errNoVault)
	}
	if err != nil {
		return domain.Vault{}, newError(server.CodeInternal, err)
	}
	return v, nil
}

func (b *Backend) getVault(ctx context.Context, id string) (domain.Vault, error) {
	v, err := b.vaults(domain.VaultID(id)).Selector.GetVault(ctx)
	if errors.Is(err, host.ErrUnknownVault) {
		return domain.Vault{}, newError(server.CodeNotFound, err)
	}
	if err != nil {
		return domain.Vault{}, newError(server.CodeInternal, err)
	}
	return v, nil
}

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
