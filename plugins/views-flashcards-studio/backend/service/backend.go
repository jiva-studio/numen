// Package service is the backend of the flashcards-studio plugin: the cards, the presets and the
// review that are served to the windows, and the tools it offers an agent. It works the vault
// through the host.
package service

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/apierror"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/hostports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/scenarios"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/sessions"
)

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
	now    ports.Clock
	cfg    Config
	runs   *sessions.Registry

	current *scenarios.Current
}

func (b *Backend) scenarios() *scenarios.Scenarios { return b.current.Get() }

// New makes the backend over the host's vault.
func New(cfg Config) *Backend {
	b := &Backend{
		vaults: cfg.Vaults, ports: hostports.New(cfg.Vaults, cfg.Store), now: cfg.Now, cfg: cfg,
		runs: sessions.NewRegistry(), current: nil,
	}
	b.current = scenarios.NewCurrent(b.getInputs(cfg.DayStart))
	return b
}

func (b *Backend) getInputs(dayStart func() time.Duration) scenarios.Inputs {
	return scenarios.Inputs{
		Readers: b.ports.Readers, Writers: b.ports.Writers, Notes: b.ports.Notes, Types: b.ports.Notes, CardModels: b.ports.Notes, Links: b.ports.Links,
		Problems: b.ports.Problems, Logs: b.ports.Logs, Schedules: b.ports.Schedules,
		Index: func(ctx context.Context, v domain.Vault, paths []string) error {
			return b.vaults(v.ID).Index.Scan(ctx, paths)
		},
		DayStart: dayStart, Now: b.now, Cores: b.cfg.Cores, Scheduler: b.cfg.Scheduler,
	}
}

func (b *Backend) getCurrentVault(ctx context.Context) (domain.Vault, error) {
	v, err := b.vaults("").Selector.GetVault(ctx)
	if err != nil {
		return domain.Vault{}, apierror.NewCurrentVaultError(err)
	}
	return v, nil
}

func (b *Backend) getVault(ctx context.Context, id string) (domain.Vault, error) {
	v, err := b.vaults(domain.VaultID(id)).Selector.GetVault(ctx)
	if err != nil {
		return domain.Vault{}, apierror.NewVaultError(err)
	}
	return v, nil
}

func (b *Backend) getTitle(ctx context.Context, v domain.Vault, path string) string {
	if path == "" {
		return ""
	}
	found, err := b.ports.Notes.GetNotes(ctx, v.ID, []string{path})
	if err != nil {
		return ""
	}
	return found[path].Title
}

func (b *Backend) newTitler(ctx context.Context, v domain.Vault) func(path string) string {
	return func(path string) string { return b.getTitle(ctx, v, path) }
}
