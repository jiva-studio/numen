package container_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/container"
)

// Every section of the settings file reaches the configuration the application
// is built from. A section left behind is a part of the application that does
// nothing, and turning a model off is done by naming none, so nothing says it.
func TestEverySectionOfTheSettingsIsCarried(t *testing.T) {
	said := settings.Defaults().Indexing
	cfg := container.Config{IndexPath: "/somewhere/index.db"}.Indexing(said)

	if !reflect.DeepEqual(cfg.Embedding, said.Embedding) {
		t.Error("the embedding section did not arrive")
	}
	if !reflect.DeepEqual(cfg.Recognition, said.Recognition.Config) {
		t.Error("the recognition section did not arrive")
	}
	if !reflect.DeepEqual(cfg.Proofreading, said.Proofreading) {
		t.Error("the proofreading section did not arrive")
	}
	if !reflect.DeepEqual(cfg.Transcription, said.Transcription.Config) {
		t.Error("the transcription section did not arrive")
	}
	if !reflect.DeepEqual(cfg.ScanProofreading, said.Recognition.Proofread) {
		t.Error("what puts a reading right did not arrive")
	}
	if !reflect.DeepEqual(cfg.TranscriptProofreading, said.Transcription.Proofread) {
		t.Error("what puts a transcript right did not arrive")
	}
	if cfg.Transcribes != said.Transcribes() {
		t.Error("whether a recording is heard without being asked did not arrive")
	}
	if cfg.TranscribesUnder != said.TranscribesUnder() {
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
	if held := reflect.TypeOf(settings.Indexing{}).NumField(); held != 6 {
		t.Errorf("indexing holds %d sections; carry the new one in Config.Indexing", held)
	}
}
