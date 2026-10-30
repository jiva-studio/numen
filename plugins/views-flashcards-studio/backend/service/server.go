package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
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

func mount(s *server.Server, vault host.Vault, store host.Store) error {
	backend := New(Config{Vault: vault, Store: store, DayStart: nil, Now: nil, Cores: 0, Scheduler: nil})
	s.SetConfigure(func(_ context.Context, values map[string]string) error {
		backend.Configure(values)
		return nil
	})
	s.SetUpdateConfig(backend.Configure)
	return backend.Mount(s)
}

// settingDayStart is the setting that defines when a day of review begins.
const settingDayStart = "day_start_time"

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
const ServiceName = "numen.flashcards"

// Mount registers the API, the tools and the commands on s.
func (b *Backend) Mount(s *server.Server) error {
	if err := mountAPI(s.NewAPI(ServiceName), b.Cards(), b.Presets(), b.Review()); err != nil {
		return err
	}
	registry, err := b.Tools()
	if err != nil {
		return err
	}
	registry.Mount(s)
	commands, err := b.Commands()
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

// CardsAPI answers the decks and card models of a vault.
type CardsAPI interface {
	CardModelsAPI
	DecksAPI
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

// PresetsAPI answers the presets that decks are scheduled by.
type PresetsAPI interface {
	PresetReadAPI
	PresetWriteAPI
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

// ReviewAPI answers the review window: the counts, the sessions and the answers.
type ReviewAPI interface {
	ReviewSessionAPI
	ReviewWatchAPI
	ReviewStatsAPI
}

// mountAPI mounts every method of the three groups on the API and returns the first failure to
// mount one.
func mountAPI(a *server.API, cardsAPI CardsAPI, presetsAPI PresetsAPI, reviewAPI ReviewAPI) error {
	handle(a, "ListCardModels", cardsAPI.ListCardModels)
	handle(a, "CreateCardModel", cardsAPI.CreateCardModel)
	handle(a, "ReadCardModel", cardsAPI.ReadCardModel)
	handle(a, "WriteCardModel", cardsAPI.WriteCardModel)
	handle(a, "RenameCardModelField", cardsAPI.RenameCardModelField)
	handle(a, "CreateDeck", cardsAPI.CreateDeck)
	handle(a, "ReadDeck", cardsAPI.ReadDeck)
	handle(a, "WriteDeck", cardsAPI.WriteDeck)

	handle(a, "GetDeckPreset", presetsAPI.GetDeckPreset)
	handle(a, "ListPresets", presetsAPI.ListPresets)
	handle(a, "CreatePreset", presetsAPI.CreatePreset)
	handle(a, "ScheduleDeck", presetsAPI.ScheduleDeck)
	handle(a, "ReadPreset", presetsAPI.ReadPreset)
	handle(a, "WritePreset", presetsAPI.WritePreset)
	handle(a, "ComputeCurve", presetsAPI.ComputeCurve)

	stream(a, "WatchCardsDue", reviewAPI.WatchCardsDue)
	handle(a, "StartSession", reviewAPI.StartSession)
	handle(a, "RecordReview", reviewAPI.RecordReview)
	handle(a, "UndoReview", reviewAPI.UndoReview)
	stream(a, "WatchReloads", reviewAPI.WatchReloads)
	handle(a, "ListReviewDays", reviewAPI.ListReviewDays)
	handle(a, "GetDeckNeighbourhood", reviewAPI.GetDeckNeighbourhood)
	handle(a, "GetVaultDeckPreset", reviewAPI.GetVaultDeckPreset)
	return a.Err()
}

// handle mounts a unary method. A body that decodes to null is an empty request, and every
// collection of the response is encoded as an array or an object.
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

// stream mounts a server-stream method with the same request and response rules as handle.
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

// Cards returns the API of the decks and card models.
func (b *Backend) Cards() CardsAPI { return &cardsHandler{Backend: b} }

// Presets returns the API of the presets.
func (b *Backend) Presets() PresetsAPI { return &presetsHandler{Backend: b} }

// Review returns the API of the review window.
func (b *Backend) Review() ReviewAPI { return &reviewHandler{Backend: b} }
