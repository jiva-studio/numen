package container_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"
)

// Every section of the settings file is carried into the application configuration.
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

// The field count makes a section added to the settings fail this test until it
// is carried. Per-section checks cannot detect a missing section, since an
// unconfigured model is simply absent.
func TestASectionAddedToTheSettingsIsCarriedToo(t *testing.T) {
	if target := reflect.TypeOf(container.Indexing{}).NumField(); target != 1 {
		t.Errorf("indexing holds %d sections; carry the new one in Config.SetSettings", target)
	}
	// The document holds the sections the file does. The ai section is carried by the
	// router given to the host, which reads it on every query.
	if targetVal := reflect.TypeOf(container.Settings{}).NumField(); targetVal != 9 {
		t.Errorf("the settings hold %d sections; carry the new one in Config.SetSettings", targetVal)
	}
}
