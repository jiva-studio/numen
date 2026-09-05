package review_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// The defaults are written once and read twice: here, and in the window, where
// a person making a preset is shown the numbers it is scheduled by. Both read
// this one corpus, and neither owns it.
const presetCorpus = "../../../protocol/testdata/presets.json"

// defaultPreset is a preset naming nothing, in the words a preset file writes
// its values in.
type defaultPreset struct {
	Goal        string         `json:"goal"`
	ByDate      string         `json:"byDate"`
	MinutesADay int            `json:"minutesADay"`
	NewADay     int            `json:"newADay"`
	ReviewsADay int            `json:"reviewsADay"`
	Retention   float64        `json:"retention"`
	Counts      string         `json:"counts"`
	Backlog     int            `json:"backlog"`
	Load        map[string]int `json:"load"`
	EvenLoad    bool           `json:"evenLoad"`
	Learned     string         `json:"learned"`
	Interval    int            `json:"interval"`
}

// presetCorpusFile is what the corpus holds.
type presetCorpusFile struct {
	Defaults defaultPreset `json:"defaults"`
}

func readPresetCorpus(t *testing.T) presetCorpusFile {
	t.Helper()
	raw, err := os.ReadFile(presetCorpus)
	if err != nil {
		t.Fatal(err)
	}
	var out presetCorpusFile
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestAPresetNamingNothingIsScheduledByWhatTheCorpusSays(t *testing.T) {
	want := readPresetCorpus(t).Defaults

	d := review.Defaults()
	byDate := ""
	if !d.By.IsZero() {
		byDate = d.By.Format(review.Named)
	}
	load := map[string]int{}
	for day, share := range d.Load {
		load[review.DayName(day)] = share
	}
	got := defaultPreset{
		Goal:        string(d.Goal),
		ByDate:      byDate,
		MinutesADay: d.MinutesADay,
		NewADay:     d.NewADay,
		ReviewsADay: d.ReviewsADay,
		Retention:   d.Retention,
		Counts:      string(d.Counts),
		Backlog:     d.Backlog,
		Load:        load,
		EvenLoad:    d.EvenLoad,
		Learned:     string(d.Rule),
		Interval:    d.Interval,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("the defaults are %+v, want %+v", got, want)
	}
}
