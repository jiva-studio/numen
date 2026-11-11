package api

import "context"

// CardModelsAPI answers the card models of a vault.
type CardModelsAPI interface {
	ListCardModels(ctx context.Context, req *ListCardModelsRequest) (*ListCardModelsResponse, error)
	CreateCardModel(ctx context.Context, req *CreateCardModelRequest) (*CreateCardModelResponse, error)
	ReadCardModel(ctx context.Context, req *ReadCardModelRequest) (*ReadCardModelResponse, error)
	WriteCardModel(ctx context.Context, req *WriteCardModelRequest) (*WriteCardModelResponse, error)
	RenameCardModelField(ctx context.Context, req *RenameCardModelFieldRequest) (*RenameCardModelFieldResponse, error)
}

// DecksAPI answers the decks of a vault.
type DecksAPI interface {
	CreateDeck(ctx context.Context, req *CreateDeckRequest) (*CreateDeckResponse, error)
	ReadDeck(ctx context.Context, req *ReadDeckRequest) (*ReadDeckResponse, error)
	WriteDeck(ctx context.Context, req *WriteDeckRequest) (*WriteDeckResponse, error)
}

// PresetReadAPI answers the reads of the presets that decks are scheduled by.
type PresetReadAPI interface {
	GetDeckPreset(ctx context.Context, req *GetDeckPresetRequest) (*GetDeckPresetResponse, error)
	ListPresets(ctx context.Context, req *ListPresetsRequest) (*ListPresetsResponse, error)
	ReadPreset(ctx context.Context, req *ReadPresetRequest) (*ReadPresetResponse, error)
	ComputeCurve(ctx context.Context, req *ComputeCurveRequest) (*ComputeCurveResponse, error)
}

// PresetWriteAPI answers the writes of the presets that decks are scheduled by.
type PresetWriteAPI interface {
	CreatePreset(ctx context.Context, req *CreatePresetRequest) (*CreatePresetResponse, error)
	WritePreset(ctx context.Context, req *WritePresetRequest) (*WritePresetResponse, error)
	ScheduleDeck(ctx context.Context, req *ScheduleDeckRequest) (*ScheduleDeckResponse, error)
}

// ReviewSessionAPI answers the sessions and the answers of a review.
type ReviewSessionAPI interface {
	StartSession(ctx context.Context, req *StartSessionRequest) (*StartSessionResponse, error)
	RecordReview(ctx context.Context, req *RecordReviewRequest) (*RecordReviewResponse, error)
	UndoReview(ctx context.Context, req *UndoReviewRequest) (*UndoReviewResponse, error)
}

// ReviewWatchAPI answers the streams of the review window.
type ReviewWatchAPI interface {
	WatchCardsDue(ctx context.Context, req *WatchCardsDueRequest, send func(*WatchCardsDueResponse) error) error
	WatchReloads(ctx context.Context, req *WatchReloadsRequest, send func(*WatchReloadsResponse) error) error
}

// ReviewStatsAPI answers the counts and the presets of the review window.
type ReviewStatsAPI interface {
	ListReviewDays(ctx context.Context, req *ListReviewDaysRequest) (*ListReviewDaysResponse, error)
	GetDeckNeighborhood(ctx context.Context, req *GetDeckNeighborhoodRequest) (*GetDeckNeighborhoodResponse, error)
	GetVaultDeckPreset(ctx context.Context, req *GetVaultDeckPresetRequest) (*GetVaultDeckPresetResponse, error)
}
