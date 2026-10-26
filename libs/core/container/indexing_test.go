package container_test

import (
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
	if cfg.IndexPath != "/somewhere/index.db" {
		t.Error("what the command line said was written over")
	}
}

// The count is here so that a section added to the settings fails this test
// until it is carried, which is the failure the sections themselves cannot
// have: an unconfigured model is a model nobody asked for.
func TestASectionAddedToTheSettingsIsCarriedToo(t *testing.T) {
	if target := reflect.TypeOf(container.Indexing{}).NumField(); target != 1 {
		t.Errorf("indexing holds %d sections; carry the new one in Config.SetSettings", target)
	}
	// The document holds the sections the file does, and the line a person is
	// left about what it says. The ai section is carried by the router the host is
	// given, which reads it at every question.
	if targetVal := reflect.TypeOf(container.Settings{}).NumField(); targetVal != 9 {
		t.Errorf("the settings hold %d sections; carry the new one in Config.SetSettings", targetVal)
	}
}
