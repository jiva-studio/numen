package api

// Rating is how well a card was recalled.
type Rating string

// Rating values.
const (
	RatingUnspecified Rating = ""
	RatingAgain       Rating = "again"
	RatingHard        Rating = "hard"
	RatingGood        Rating = "good"
	RatingEasy        Rating = "easy"
)

// VaultCardsDue is what one vault has due for review. Unread holds the reason the
// vault could not be counted.
type VaultCardsDue struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	ReviewCards int32             `json:"reviewCards"`
	Due         int32             `json:"due"`
	New         int32             `json:"new"`
	Decks       []*DeckCardsDue   `json:"decks"`
	Presets     []*PresetCardsDue `json:"presets"`
	Unread      string            `json:"unread"`
	IsIndexing  bool              `json:"isIndexing"`
}

// DeckCardsDue is what one deck has due for review.
type DeckCardsDue struct {
	Deck        string `json:"deck"`
	ReviewCards int32  `json:"reviewCards"`
	Due         int32  `json:"due"`
	New         int32  `json:"new"`
	Reviewed    int32  `json:"reviewed"`
	Learned     int32  `json:"learned"`
	Unbegun     int32  `json:"unbegun"`
}

// PresetCardsDue is the review count of the decks of one preset and what the current day has used.
type PresetCardsDue struct {
	Preset         string `json:"preset"`
	Title          string `json:"title"`
	Decks          int32  `json:"decks"`
	Cards          int32  `json:"cards"`
	PendingReviews int32  `json:"pendingReviews"`
	PendingNew     int32  `json:"pendingNew"`
	Reviewed       int32  `json:"reviewed"`
	// DurationMs is the time spent reviewing in the current day in milliseconds.
	DurationMs      int64      `json:"durationMs"`
	ReviewedNew     int32      `json:"reviewedNew"`
	ReviewedReviews int32      `json:"reviewedReviews"`
	New             int32      `json:"new"`
	Reviews         int32      `json:"reviews"`
	Minutes         float64    `json:"minutes"`
	ClosesNew       string     `json:"closesNew"`
	ClosesReviews   string     `json:"closesReviews"`
	ClosesMinutes   string     `json:"closesMinutes"`
	ClosesBacklog   string     `json:"closesBacklog"`
	StopsOn         StopReason `json:"stopsOn"`
}

// Intervals is the interval in seconds that each rating would give a card.
type Intervals struct {
	Again int64 `json:"again"`
	Hard  int64 `json:"hard"`
	Good  int64 `json:"good"`
	Easy  int64 `json:"easy"`
}

// SessionCard is one card of a review session, laid out for one template.
type SessionCard struct {
	Deck      string     `json:"deck"`
	Section   string     `json:"section"`
	CardID    string     `json:"cardId"`
	Template  string     `json:"template"`
	Heading   string     `json:"heading"`
	Front     string     `json:"front"`
	Back      string     `json:"back"`
	IsNew     bool       `json:"isNew"`
	Due       string     `json:"due"`
	Intervals *Intervals `json:"intervals,omitempty"`
}

// WatchCardsDueRequest is the request of the WatchCardsDue method.
type WatchCardsDueRequest struct {
}

// WatchCardsDueResponse is the response of the WatchCardsDue method.
type WatchCardsDueResponse struct {
	// Vaults lists every known vault; sent first and again while no count has arrived.
	Vaults []*VaultCardsDue `json:"vaults"`
	Day    string           `json:"day"`
	// Counted is the count of one vault, sent as soon as it is worked out.
	Counted *VaultCardsDue `json:"counted,omitempty"`
}

// StartSessionRequest is the request of the StartSession method.
type StartSessionRequest struct {
	Vault string `json:"vault"`
	Deck  string `json:"deck"`
	// Preset selects the decks of one preset; the path may be empty for the default preset. Absent selects no preset.
	Preset *string `json:"preset,omitempty"`
}

// StartSessionResponse is the response of the StartSession method.
type StartSessionResponse struct {
	Run       string         `json:"run"`
	Cards     []*SessionCard `json:"cards"`
	Unwritten []string       `json:"unwritten"`
	Skipped   int32          `json:"skipped"`
}

// RecordReviewRequest is the request of the RecordReview method.
type RecordReviewRequest struct {
	Vault    string `json:"vault"`
	Run      string `json:"run"`
	CardID   string `json:"cardId"`
	Template string `json:"template"`
	Rating   Rating `json:"rating"`
	// DurationMs is the time spent on the card in milliseconds.
	DurationMs int64 `json:"durationMs"`
}

// RecordReviewResponse is the response of the RecordReview method.
type RecordReviewResponse struct {
	Answer string `json:"answer"`
}

// UndoReviewRequest is the request of the UndoReview method.
type UndoReviewRequest struct {
	Vault  string `json:"vault"`
	Run    string `json:"run"`
	Answer string `json:"answer"`
}

// UndoReviewResponse is the response of the UndoReview method.
type UndoReviewResponse struct {
}

// ListReviewDaysRequest is the request of the ListReviewDays method.
type ListReviewDaysRequest struct {
	Vault string `json:"vault"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// ListReviewDaysResponse is the response of the ListReviewDays method.
type ListReviewDaysResponse struct {
	Days     []*ReviewDay `json:"days"`
	Due      []*ReviewDay `json:"due"`
	Streak   int32        `json:"streak"`
	Reviewed int32        `json:"reviewed"`
}

// ReviewDay is the review counts of one day. Tested and Recalled count the cards whose retention was measured.
type ReviewDay struct {
	Day      string `json:"day"`
	Reviewed int32  `json:"reviewed"`
	Again    int32  `json:"again"`
	Hard     int32  `json:"hard"`
	Good     int32  `json:"good"`
	Easy     int32  `json:"easy"`
	Tested   int32  `json:"tested"`
	Recalled int32  `json:"recalled"`
}

// GetDeckNeighbourhoodRequest is the request of the GetDeckNeighbourhood method.
type GetDeckNeighbourhoodRequest struct {
	Vault string `json:"vault"`
	Deck  string `json:"deck"`
}

// GetDeckNeighbourhoodResponse is the response of the GetDeckNeighbourhood method.
type GetDeckNeighbourhoodResponse struct {
	Notes  []*DeckNeighbour `json:"notes"`
	Unread int32            `json:"unread"`
}

// DeckNeighbour is a note a deck links to or is linked from. Address is the link target as the deck spells it.
type DeckNeighbour struct {
	Address     string    `json:"address"`
	Path        string    `json:"path"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	Label       string    `json:"label"`
	IsPointedAt bool      `json:"isPointedAt"`
	IsAmbiguous bool      `json:"isAmbiguous"`
	Error       ErrorCode `json:"error,omitempty"`
}

// WatchReloadsRequest is the request of the WatchReloads method.
type WatchReloadsRequest struct {
}

// WatchReloadsResponse is the response of the WatchReloads method.
type WatchReloadsResponse struct {
	// Reload is true when the vault changed; false messages only show the stream is open.
	Reload bool `json:"reload"`
}

// GetVaultDeckPresetRequest is the request of the GetVaultDeckPreset method.
type GetVaultDeckPresetRequest struct {
	Vault string `json:"vault"`
	Deck  string `json:"deck"`
}

// GetVaultDeckPresetResponse is the response of the GetVaultDeckPreset method.
type GetVaultDeckPresetResponse struct {
	Preset *Preset   `json:"preset,omitempty"`
	Error  ErrorCode `json:"error,omitempty"`
}
