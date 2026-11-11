// Package cachefile is the shape of the cache files that spare a reading of the answers: the days
// counted from each run, and the schedules worked out from the runs.
package cachefile

import (
	"encoding/json"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
)

// Counts is the cache of the counted days, kept by run.
type Counts struct {
	V    int   `json:"v"`
	Runs []Run `json:"runs"`
}

// Run is what was counted from one run file of the answers.
type Run struct {
	Name string                  `json:"name"`
	Size int                     `json:"size"`
	Days map[string]review.Tally `json:"days"`
	IDs  []string                `json:"ids"`
}

// Schedules is the cache of the schedules, kept with the runs it was worked out from.
type Schedules struct {
	V           int        `json:"v"`
	By          string     `json:"by"`
	Files       []File     `json:"files"`
	ReviewCards []Schedule `json:"templates"`
}

// File is a run file a schedule cache was worked out from.
type File struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

// Schedule is where the answers left one review card.
type Schedule struct {
	Card       string  `json:"card"`
	Template   string  `json:"template"`
	Due        string  `json:"due"`
	Last       string  `json:"last"`
	Reps       int     `json:"reps"`
	Lapses     int     `json:"lapses"`
	Stability  float64 `json:"stability"`
	Difficulty float64 `json:"difficulty"`
	Phase      uint8   `json:"phase"`
}

// DecodeCounts reads a cache of the counted days.
func DecodeCounts(raw []byte) (Counts, error) {
	var cache Counts
	err := json.Unmarshal(raw, &cache)
	return cache, err
}

// EncodeCounts returns the cache of the counted days as it is kept.
func EncodeCounts(cache Counts) ([]byte, error) {
	return json.Marshal(cache)
}

// DecodeSchedules reads a cache of the schedules.
func DecodeSchedules(raw []byte) (Schedules, error) {
	var cache Schedules
	err := json.Unmarshal(raw, &cache)
	return cache, err
}

// EncodeSchedules returns the cache of the schedules as it is kept.
func EncodeSchedules(cache Schedules) ([]byte, error) {
	return json.Marshal(cache)
}
