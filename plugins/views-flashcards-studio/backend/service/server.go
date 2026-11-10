package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/hostports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// Mount registers the services of the plugin on s, working the vault and the plugin's folder of its
// store through the host the environment names.
func Mount(s *server.Server) error {
	client, err := host.NewClientFromEnv()
	if err != nil {
		return err
	}
	store, err := host.NewStoreClientFromEnv()
	if err != nil {
		return err
	}
	return mount(s, client, store)
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given. It works the vault through the host
// those name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		client, err := host.NewClientFromVars(vars)
		if err != nil {
			return err
		}
		store, err := host.NewStoreClientFromVars(vars)
		if err != nil {
			return err
		}
		return mount(s, client, store)
	})
}

func mount(s *server.Server, client *host.Client, store *host.StoreClient) error {
	folders := func(id domain.VaultID) hostports.Folder { return store.With(id) }
	backend := New(Config{Vaults: vaultsOf(client), Store: folders, DayStart: nil, Now: clock.System{}, Cores: 0, Scheduler: nil})
	s.SetConfigure(func(_ context.Context, values map[string]string, _ server.Secrets) error {
		backend.Configure(values)
		return nil
	})
	s.SetUpdateConfig(func(values map[string]string, _ server.Secrets) { backend.Configure(values) })
	return backend.Mount(s)
}

// settingDayStart is the setting that defines when a day of review begins.
const settingDayStart = "dayStartTime"

// Configure applies the settings supplied by the host. A day start that is not a
// valid hour uses the default.
func (b *Backend) Configure(values map[string]string) {
	starts, err := review.ParseDayStart(values[settingDayStart])
	if err != nil {
		fmt.Fprintln(os.Stderr, "flashcards-studio:", settingDayStart, err)
	}
	b.Reconfigure(func() time.Duration { return starts })
}

// ServiceName is the name of the API service the backend serves; the manifest declares it.
const ServiceName = "md.numen.views.flashcards.studio"

// Mount registers the API, the tools and the commands on s.
func (b *Backend) Mount(s *server.Server) error {
	// gomutants:disable-next-line BRANCH_IF reason="Static service and method names are always valid."
	if err := mountAPI(
		s.NewAPI(ServiceName), b.CardModels(), b.Decks(), b.PresetReads(), b.PresetWrites(),
		b.ReviewSessions(), b.ReviewWatches(), b.ReviewStats(),
	); err != nil {
		return err
	}
	registry, err := b.Tools()
	// gomutants:disable-next-line BRANCH_IF reason="Static tool schemas are always valid."
	if err != nil {
		return err
	}
	registry.Mount(s)
	commands, err := b.Commands()
	// gomutants:disable-next-line BRANCH_IF reason="Static command registrations are always valid."
	if err != nil {
		return err
	}
	command.Mount(s, commands)
	return nil
}

// CardModelsAPI answers the card models of a vault.
type CardModelsAPI interface {
	ListCardModels(ctx context.Context, req *api.ListCardModelsRequest) (*api.ListCardModelsResponse, error)
	CreateCardModel(ctx context.Context, req *api.CreateCardModelRequest) (*api.CreateCardModelResponse, error)
	ReadCardModel(ctx context.Context, req *api.ReadCardModelRequest) (*api.ReadCardModelResponse, error)
	WriteCardModel(ctx context.Context, req *api.WriteCardModelRequest) (*api.WriteCardModelResponse, error)
	RenameCardModelField(ctx context.Context, req *api.RenameCardModelFieldRequest) (*api.RenameCardModelFieldResponse, error)
}

// DecksAPI answers the decks of a vault.
type DecksAPI interface {
	CreateDeck(ctx context.Context, req *api.CreateDeckRequest) (*api.CreateDeckResponse, error)
	ReadDeck(ctx context.Context, req *api.ReadDeckRequest) (*api.ReadDeckResponse, error)
	WriteDeck(ctx context.Context, req *api.WriteDeckRequest) (*api.WriteDeckResponse, error)
}

// PresetReadAPI answers the reads of the presets that decks are scheduled by.
type PresetReadAPI interface {
	GetDeckPreset(ctx context.Context, req *api.GetDeckPresetRequest) (*api.GetDeckPresetResponse, error)
	ListPresets(ctx context.Context, req *api.ListPresetsRequest) (*api.ListPresetsResponse, error)
	ReadPreset(ctx context.Context, req *api.ReadPresetRequest) (*api.ReadPresetResponse, error)
	ComputeCurve(ctx context.Context, req *api.ComputeCurveRequest) (*api.ComputeCurveResponse, error)
}

// PresetWriteAPI answers the writes of the presets that decks are scheduled by.
type PresetWriteAPI interface {
	CreatePreset(ctx context.Context, req *api.CreatePresetRequest) (*api.CreatePresetResponse, error)
	WritePreset(ctx context.Context, req *api.WritePresetRequest) (*api.WritePresetResponse, error)
	ScheduleDeck(ctx context.Context, req *api.ScheduleDeckRequest) (*api.ScheduleDeckResponse, error)
}

// ReviewSessionAPI answers the sessions and the answers of a review.
type ReviewSessionAPI interface {
	StartSession(ctx context.Context, req *api.StartSessionRequest) (*api.StartSessionResponse, error)
	RecordReview(ctx context.Context, req *api.RecordReviewRequest) (*api.RecordReviewResponse, error)
	UndoReview(ctx context.Context, req *api.UndoReviewRequest) (*api.UndoReviewResponse, error)
}

// ReviewWatchAPI answers the streams of the review window.
type ReviewWatchAPI interface {
	WatchCardsDue(ctx context.Context, req *api.WatchCardsDueRequest, send func(*api.WatchCardsDueResponse) error) error
	WatchReloads(ctx context.Context, req *api.WatchReloadsRequest, send func(*api.WatchReloadsResponse) error) error
}

// ReviewStatsAPI answers the counts and the presets of the review window.
type ReviewStatsAPI interface {
	ListReviewDays(ctx context.Context, req *api.ListReviewDaysRequest) (*api.ListReviewDaysResponse, error)
	GetDeckNeighbourhood(ctx context.Context, req *api.GetDeckNeighbourhoodRequest) (*api.GetDeckNeighbourhoodResponse, error)
	GetVaultDeckPreset(ctx context.Context, req *api.GetVaultDeckPresetRequest) (*api.GetVaultDeckPresetResponse, error)
}

func mountAPI(
	a *server.API, models CardModelsAPI, decks DecksAPI, presetReads PresetReadAPI, presetWrites PresetWriteAPI,
	sessions ReviewSessionAPI, watches ReviewWatchAPI, stats ReviewStatsAPI,
) error {
	handle(a, "ListCardModels", models.ListCardModels)
	handle(a, "CreateCardModel", models.CreateCardModel)
	handle(a, "ReadCardModel", models.ReadCardModel)
	handle(a, "WriteCardModel", models.WriteCardModel)
	handle(a, "RenameCardModelField", models.RenameCardModelField)
	handle(a, "CreateDeck", decks.CreateDeck)
	handle(a, "ReadDeck", decks.ReadDeck)
	handle(a, "WriteDeck", decks.WriteDeck)

	handle(a, "GetDeckPreset", presetReads.GetDeckPreset)
	handle(a, "ListPresets", presetReads.ListPresets)
	handle(a, "CreatePreset", presetWrites.CreatePreset)
	handle(a, "ScheduleDeck", presetWrites.ScheduleDeck)
	handle(a, "ReadPreset", presetReads.ReadPreset)
	handle(a, "WritePreset", presetWrites.WritePreset)
	handle(a, "ComputeCurve", presetReads.ComputeCurve)

	stream(a, "WatchCardsDue", watches.WatchCardsDue)
	handle(a, "StartSession", sessions.StartSession)
	handle(a, "RecordReview", sessions.RecordReview)
	handle(a, "UndoReview", sessions.UndoReview)
	stream(a, "WatchReloads", watches.WatchReloads)
	handle(a, "ListReviewDays", stats.ListReviewDays)
	handle(a, "GetDeckNeighbourhood", stats.GetDeckNeighbourhood)
	handle(a, "GetVaultDeckPreset", stats.GetVaultDeckPreset)
	return a.Err()
}

func handle[Req, Res any](a *server.API, method string, fn func(context.Context, *Req) (*Res, error)) {
	server.Handle(a, method, func(ctx context.Context, req *Req) (*Res, error) {
		if req == nil {
			req = new(Req)
		}
		res, err := fn(ctx, req)
		if err != nil {
			return nil, err
		}
		api.FillCollections(res)
		return res, nil
	})
}

func stream[Req, Res any](
	a *server.API, method string, fn func(context.Context, *Req, func(*Res) error) error,
) {
	server.Stream(a, method, func(ctx context.Context, req *Req, send func(*Res) error) error {
		if req == nil {
			req = new(Req)
		}
		return fn(ctx, req, func(res *Res) error {
			api.FillCollections(res)
			return send(res)
		})
	})
}

// CardModels returns the API of the card models.
func (b *Backend) CardModels() CardModelsAPI { return &cardsHandler{Backend: b} }

// Decks returns the API of the decks.
func (b *Backend) Decks() DecksAPI { return &cardsHandler{Backend: b} }

// PresetReads returns the API of the reads of the presets.
func (b *Backend) PresetReads() PresetReadAPI { return &presetsHandler{Backend: b} }

// PresetWrites returns the API of the writes of the presets.
func (b *Backend) PresetWrites() PresetWriteAPI { return &presetsHandler{Backend: b} }

// ReviewSessions returns the API of the sessions and the answers of a review.
func (b *Backend) ReviewSessions() ReviewSessionAPI { return &reviewHandler{Backend: b} }

// ReviewWatches returns the API of the streams of the review window.
func (b *Backend) ReviewWatches() ReviewWatchAPI { return &reviewHandler{Backend: b} }

// ReviewStats returns the API of the counts and the presets of the review window.
func (b *Backend) ReviewStats() ReviewStatsAPI { return &reviewHandler{Backend: b} }

// vaultsOf gives the ports' calls on the vault of the client with an identity; the empty identity is the
// vault the host shows.
func vaultsOf(client *host.Client) func(domain.VaultID) hostports.Vault {
	return func(id domain.VaultID) hostports.Vault {
		on := client.With(id)
		return hostports.Vault{Files: on, Writer: on, Index: on, Notes: on, Links: on, Cache: on, Selector: on}
	}
}
