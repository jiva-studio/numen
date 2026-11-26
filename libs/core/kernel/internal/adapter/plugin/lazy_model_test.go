package plugin_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"
)

const speechModel = "speech-model"

func newRuntimeAndModelPlugin(t *testing.T, source *modelSource) lazyPlugin {
	t.Helper()
	t.Setenv(source.envName(sampleRuntime), "")
	t.Setenv(source.envName(speechModel), "")
	data := []byte("speech model bytes of an example")
	manifest := newModelManifest(fmt.Sprintf(
		`{"id": %q, "kind": "runtime"}, {"id": %q, "kind": "model", "version": "1", "sourceUrl": "https://models.invalid/speech.bin", "checksumSha256": %q, "sizeBytes": %d}`,
		sampleRuntime, speechModel, sumHex(data), len(data),
	))
	return newLazyPluginFrom(t, provision.NewObservedRegistry(nil, source), manifest)
}

func TestTheUseOfAPluginProvisionsItsRuntimeAndNotItsModels(t *testing.T) {
	source := newModelSource(t)
	p := newRuntimeAndModelPlugin(t, source)

	if err := p.use(t); err != nil {
		t.Fatal(err)
	}

	if got := source.count(); got != 1 {
		t.Errorf("the use provisioned %d dependencies, want the runtime only", got)
	}
	if _, err := os.Stat(source.path(sampleRuntime)); err != nil {
		t.Errorf("the runtime is not there after the use: %v", err)
	}
	if _, err := os.Stat(source.path(speechModel)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the use fetched the model: %v", err)
	}
}

func TestAPluginIsLaunchedWithoutAVariableForItsModels(t *testing.T) {
	source := newModelSource(t)
	p := newRuntimeAndModelPlugin(t, source)

	if err := p.use(t); err != nil {
		t.Fatal(err)
	}

	vars := p.backend.lastVars(t)
	if _, ok := vars[source.envName(speechModel)]; ok {
		t.Errorf("the plugin was launched with %s = %q", source.envName(speechModel), vars[source.envName(speechModel)])
	}
	if got := vars[source.envName(sampleRuntime)]; got != source.path(sampleRuntime) {
		t.Errorf("the plugin was launched with %s = %q, want %q", source.envName(sampleRuntime), got, source.path(sampleRuntime))
	}
}
