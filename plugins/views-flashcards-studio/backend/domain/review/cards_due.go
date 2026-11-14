package review

import "time"

// CardsDue contains vault card counts, due cards, and unreviewed card statistics for the current day.
type CardsDue struct {
	ReviewCards int
	Due         int
	New         int
	Decks       []DeckCardsDue
	Presets     []PresetCardsDue
}

// DeckCardsDue tracks card counts and review status for a single deck.
type DeckCardsDue struct {
	Deck        string
	ReviewCards int
	Due         int
	New         int
	Unbegun     int
	Answered    int
	Learned     int
}

// PresetCardsDue contains daily budget and review metrics for a preset.
type PresetCardsDue struct {
	Preset          string
	Decks           int
	Cards           int
	Due             int
	New             int
	Answered        int
	ReviewedNew     int
	ReviewedReviews int
	Took            time.Duration
	Budget          Budget
	Limits          Limits
	Stops           StopReason
}
