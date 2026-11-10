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
	if err := mountAPI(s.NewAPI(ServiceName), apiHandlers{
		Models: b.CardModels(), Decks: b.Decks(), PresetReads: b.PresetReads(), PresetWrites: b.PresetWrites(),
		Sessions: b.ReviewSessions(), Watches: b.ReviewWatches(), Stats: b.ReviewStats(),
	}); err != nil {
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

type apiHandlers struct {
	Models       CardModelsAPI
	Decks        DecksAPI
	PresetReads  PresetReadAPI
	PresetWrites PresetWriteAPI
	Sessions     ReviewSessionAPI
	Watches      ReviewWatchAPI
	Stats        ReviewStatsAPI
}

func mountAPI(a *server.API, h apiHandlers) error {
	handle(a, "ListCardModels", h.Models.ListCardModels)
	handle(a, "CreateCardModel", h.Models.CreateCardModel)
	handle(a, "ReadCardModel", h.Models.ReadCardModel)
	handle(a, "WriteCardModel", h.Models.WriteCardModel)
	handle(a, "RenameCardModelField", h.Models.RenameCardModelField)
	handle(a, "CreateDeck", h.Decks.CreateDeck)
	handle(a, "ReadDeck", h.Decks.ReadDeck)
	handle(a, "WriteDeck", h.Decks.WriteDeck)

	handle(a, "GetDeckPreset", h.PresetReads.GetDeckPreset)
	handle(a, "ListPresets", h.PresetReads.ListPresets)
	handle(a, "CreatePreset", h.PresetWrites.CreatePreset)
	handle(a, "ScheduleDeck", h.PresetWrites.ScheduleDeck)
	handle(a, "ReadPreset", h.PresetReads.ReadPreset)
	handle(a, "WritePreset", h.PresetWrites.WritePreset)
	handle(a, "ComputeCurve", h.PresetReads.ComputeCurve)

	stream(a, "WatchCardsDue", h.Watches.WatchCardsDue)
	handle(a, "StartSession", h.Sessions.StartSession)
	handle(a, "RecordReview", h.Sessions.RecordReview)
	handle(a, "UndoReview", h.Sessions.UndoReview)
	stream(a, "WatchReloads", h.Watches.WatchReloads)
	handle(a, "ListReviewDays", h.Stats.ListReviewDays)
	handle(a, "GetDeckNeighbourhood", h.Stats.GetDeckNeighbourhood)
	handle(a, "GetVaultDeckPreset", h.Stats.GetVaultDeckPreset)
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

func vaultsOf(client *host.Client) func(domain.VaultID) hostports.Vault {
	return func(id domain.VaultID) hostports.Vault {
		on := client.With(id)
		return hostports.Vault{Files: on, Writer: on, Index: on, Notes: on, Links: on, Cache: on, Selector: on}
	}
}
