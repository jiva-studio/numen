// Package service is the backend of the flashcards-studio plugin: the cards, the presets and the
// review that are served to the windows, and the tools it offers an agent. It works the vault
// through the host.
package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/hostclient"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/hostports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/presets"
)

// Config is what the backend is made from.
type Config struct {
	GetVault  func(domain.VaultID) ports.HostVault
	Store     func(domain.VaultID) ports.HostFolder
	Now       ports.Clock
	Cores     int
	Scheduler ports.ScheduleAlgorithm
}

type flashReview = flashcards.UseCases[presets.UseCases]

// Backend answers the cards, the presets and the review.
type Backend struct {
	*vaultSession
	ports   hostports.Ports
	now     ports.Clock
	entropy ports.Entropy
	cfg     Config
	runs    *flashcards.RunRegistry
	flash   *flashcards.Current[presets.UseCases]
}

// New makes the backend over the host's vault.
func New(cfg Config) *Backend {
	hostPorts := hostports.New(cfg.GetVault, cfg.Store)
	b := &Backend{
		vaultSession: &vaultSession{getHostVault: cfg.GetVault, notes: hostPorts.Notes}, ports: hostPorts,
		now: cfg.Now, entropy: clock.Entropy{}, cfg: cfg, runs: flashcards.NewRunRegistry(), flash: nil,
	}
	b.flash = flashcards.NewCurrent(b.getFlashConfig(), b.newPresets)
	return b
}

func (b *Backend) updateIndex(ctx context.Context, v domain.Vault, paths []string) error {
	return b.getHostVault(v.ID).Index.Scan(ctx, paths)
}

func (b *Backend) getFlashConfig() flashcards.Config {
	c := b.getCardsConfig()
	return flashcards.Config{
		Readers: c.Readers, Notes: b.ports.Notes, Cards: cards.NewRead(c.Readers, c.Links), NewDeckWriter: c.NewDeckWriter,
		Index: b.updateIndex, Logs: b.ports.Logs, Schedules: b.ports.GetScheduleStore("schedules"), DayCounts: b.ports.GetScheduleStore("days"),
		Report: hostclient.ReportError, Day: review.Day{Starts: 0, In: nil}, Now: b.now, Entropy: b.entropy, Cores: b.cfg.Cores,
		Scheduler: b.cfg.Scheduler,
	}
}

func (b *Backend) newPresets(index flashcards.IndexUpdate, day review.Day) (presets.UseCases, flashcards.OpenPresetReads) {
	cfg := presets.Config{Readers: b.ports.Readers, Writers: b.ports.Writers, Links: b.ports.Links, Notes: b.ports.Notes,
		Problems: b.ports.Problems, Index: index, Now: b.now, Day: day, LinkWriter: b.getCardsConfig().NewLinkWriter(index)}
	return presets.NewUseCases(cfg), presets.NewOpenReads(cfg)
}

func (b *Backend) getCardsConfig() cards.Config {
	return cards.Config{
		Readers: b.ports.Readers, Writers: b.ports.Writers, Notes: b.ports.Notes, CardModels: b.ports.Notes, Links: b.ports.Links,
		Index: b.updateIndex, Now: b.now, Entropy: b.entropy,
	}
}
