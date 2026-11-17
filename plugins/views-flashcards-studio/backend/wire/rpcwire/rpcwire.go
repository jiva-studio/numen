// Package rpcwire registers the JSON methods of the plugin on the API of the SDK server.
package rpcwire

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
)

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

// Handlers answers every method of the service.
type Handlers struct {
	Models       api.CardModelsAPI
	Decks        api.DecksAPI
	PresetReads  api.PresetReadAPI
	PresetWrites api.PresetWriteAPI
	Sessions     api.ReviewSessionAPI
	Watches      api.ReviewWatchAPI
	Stats        api.ReviewStatsAPI
}

// Mount registers every method of the service on a.
func Mount(a *server.API, h Handlers) error {
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
	handle(a, "GetDeckNeighborhood", h.Stats.GetDeckNeighborhood)
	handle(a, "GetVaultDeckPreset", h.Stats.GetVaultDeckPreset)
	return a.Err()
}
