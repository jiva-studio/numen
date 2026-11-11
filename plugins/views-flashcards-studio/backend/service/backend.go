// Package service is the backend of the flashcards-studio plugin: the cards, the presets and the
// review that are served to the windows, and the tools it offers an agent. It works the vault
// through the host.
package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/hostclient"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/hostports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/scenarios"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/sessions"
)

// Config is what the backend is made from.
type Config struct {
	Vaults    func(domain.VaultID) ports.HostVault
	Store     func(domain.VaultID) ports.HostFolder
	Now       ports.Clock
	Cores     int
	Scheduler ports.ScheduleAlgorithm
}

// Backend answers the cards, the presets and the review.
type Backend struct {
	*session
	ports hostports.Ports
	now   ports.Clock
	cfg   Config
	runs  *sessions.Registry
	flash *scenarios.Current
}

// New makes the backend over the host's vault.
func New(cfg Config) *Backend {
	bound := hostports.New(cfg.Vaults, cfg.Store)
	b := &Backend{
		session: &session{vaults: cfg.Vaults, notes: bound.Notes}, ports: bound, now: cfg.Now, cfg: cfg,
		runs: sessions.NewRegistry(), flash: nil,
	}
	b.flash = scenarios.NewCurrent(b.getFlashConfig())
	return b
}

func (b *Backend) updateIndex(ctx context.Context, v domain.Vault, paths []string) error {
	return b.vaults(v.ID).Index.Scan(ctx, paths)
}

// getFlashConfig is what the review is built over; the host's cache keeps the schedules and the days counted.
func (b *Backend) getFlashConfig() flashcards.Config {
	return flashcards.Config{
		Readers: b.ports.Readers, Writers: b.ports.Writers, Notes: b.ports.Notes, Links: b.ports.Links,
		Problems: b.ports.Problems, Index: b.updateIndex, Logs: b.ports.Logs,
		Schedules: b.ports.Schedules("schedules"), DayCounts: b.ports.Schedules("days"),
		Report: hostclient.ReportError, Day: review.Day{Starts: 0, In: nil}, Now: b.now, Cores: b.cfg.Cores,
		Scheduler: b.cfg.Scheduler,
	}
}

func (b *Backend) getCardsConfig() cards.Config {
	return cards.Config{
		Readers: b.ports.Readers, Writers: b.ports.Writers, Notes: b.ports.Notes, CardModels: b.ports.Notes, Links: b.ports.Links,
		Index: b.updateIndex, Now: b.now,
	}
}
