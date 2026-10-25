package container_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"
)

// Every section of the settings file reaches the configuration the application
// nothing, and turning a model off is done by naming none, so nothing says it.
func TestEverySectionOfTheSettingsIsCarried(t *testing.T) {
	response := container.DefaultIndexing()
	target := container.Settings{Indexing: response}
	cfg := container.Config{IndexPath: "/somewhere/index.db"}.SetSettings(target)

	if !reflect.DeepEqual(cfg.Embedding, response.Embedding) {
		t.Error("the embedding section did not arrive")
	}
	if !reflect.DeepEqual(cfg.Proofreading, response.Proofreading) {
		t.Error("the proofreading section did not arrive")
	}
	if !reflect.DeepEqual(cfg.ScanProofreading, response.Recognition.Proofread) {
		t.Error("what puts a recognition right did not arrive")
	}
	if !reflect.DeepEqual(cfg.TranscriptProofreading, response.Transcription.Proofread) {
		t.Error("what puts a transcript right did not arrive")
	}
	if cfg.ShouldTranscribe != response.CanTranscribe() {
		t.Error("whether a recording is heard without being asked did not arrive")
	}
	if cfg.TranscribesUnder != response.GetTranscribeLimit() {
		t.Error("how large a recording heard unasked may be did not arrive")
	}
	if cfg.IndexPath != "/somewhere/index.db" {
		t.Error("what the command line said was written over")
	}
}

// The count is here so that a section added to the settings fails this test
// until it is carried, which is the failure the sections themselves cannot
// have: an unconfigured model is a model nobody asked for.
func TestASectionAddedToTheSettingsIsCarriedToo(t *testing.T) {
	if target := reflect.TypeOf(container.Indexing{}).NumField(); target != 6 {
		t.Errorf("indexing holds %d sections; carry the new one in Config.SetSettings", target)
	}
	// The document holds the sections the file does, and the line a person is
	// left about what it says. The ai section is carried by the router the host is
	// given, which reads it at every question.
	if targetVal := reflect.TypeOf(container.Settings{}).NumField(); targetVal != 10 {
		t.Errorf("the settings hold %d sections; carry the new one in Config.SetSettings", targetVal)
	}
}

// A section naming nothing listens to recordings; one naming a choice follows it.
func TestARecordingIsListenedToUnlessTheSettingsSayNot(t *testing.T) {
	yes, no := true, false
	for _, one := range []struct {
		what string
		set  *bool
		want bool
	}{
		{"nothing named", nil, true},
		{"yes", &yes, true},
		{"no", &no, false},
	} {
		if got := (container.Indexing{TranscribeRecordings: one.set}).CanTranscribe(); got != one.want {
			t.Errorf("%s: %v, want %v", one.what, got, one.want)
		}
	}
}

// The size a recording may run to is read in megabytes: none named is 300, a
// negative number is none at all, and a number too large to count in bytes is
// no limit.
func TestTheLimitOnARecordingIsReadInMegabytes(t *testing.T) {
	const largest = int(math.MaxInt64 >> 20)
	for _, one := range []struct {
		what string
		mb   int
		want int64
	}{
		{"none named", 0, 300 * 1024 * 1024},
		{"negative", -1, 0},
		{"very negative", -500, 0},
		{"one", 1, 1024 * 1024},
		{"ten", 10, 10 * 1024 * 1024},
		{"the most that is counted", largest, int64(largest) * 1024 * 1024},
		{"one past the most", largest + 1, math.MaxInt64},
	} {
		got := container.Indexing{TranscribeUnderMB: one.mb}.GetTranscribeLimit()
		if got != one.want {
			t.Errorf("%s: %d bytes, want %d", one.what, got, one.want)
		}
	}
}
