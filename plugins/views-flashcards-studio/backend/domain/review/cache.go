package review

import "encoding/json"

// CountsCache is the cache of the counted days, kept by run. It spares a reading of the answers.
type CountsCache struct {
	V    int        `json:"v"`
	Runs []RunCache `json:"runs"`
}

// RunCache is what was counted from one run file of the answers.
type RunCache struct {
	Name string           `json:"name"`
	Size int              `json:"size"`
	Days map[string]Tally `json:"days"`
	IDs  []string         `json:"ids"`
}

// SchedulesCache is the cache of the schedules, kept with the runs it was worked out from.
type SchedulesCache struct {
	V           int             `json:"v"`
	By          string          `json:"by"`
	Files       []FileCache     `json:"files"`
	ReviewCards []ScheduleCache `json:"templates"`
}

// FileCache is a run file a schedule cache was worked out from.
type FileCache struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

// ScheduleCache is where the answers left one review card.
type ScheduleCache struct {
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
func DecodeCounts(raw []byte) (CountsCache, error) {
	var cache CountsCache
	err := json.Unmarshal(raw, &cache)
	return cache, err
}

// EncodeCounts returns the cache of the counted days as it is kept.
func EncodeCounts(cache CountsCache) ([]byte, error) {
	return json.Marshal(cache)
}

func DecodeSchedules(raw []byte) (SchedulesCache, error) {
	var cache SchedulesCache
	err := json.Unmarshal(raw, &cache)
	return cache, err
}

// EncodeSchedules returns the cache of the schedules as it is kept.
func EncodeSchedules(cache SchedulesCache) ([]byte, error) {
	return json.Marshal(cache)
}
