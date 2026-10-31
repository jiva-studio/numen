package manifestjson_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

func TestManifestDeclaresARuntimeAsADependency(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},
		"dependencies":[{"id":"sample-runtime","kind":"runtime"}]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(m.GetRuntimeIDs(), []string{"sample-runtime"}) {
		t.Errorf("runtime ids = %v, want [sample-runtime]", m.GetRuntimeIDs())
	}
	got := m.GetDependencyRequirements("/p", nil)
	if len(got) != 1 || got[0].Kind != manifest.DependencyKindRuntime || got[0].PluginDir != "/p" {
		t.Errorf("requirements = %+v", got)
	}
}

func TestManifestNoLongerReadsARuntimesList(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","runtimes":["sample-runtime"],"backend":{"entrypoint":"bin/a"}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.GetRuntimeIDs()) != 0 || len(m.Warnings) != 1 || !strings.Contains(m.Warnings[0], "runtimes") {
		t.Errorf("runtime ids = %v, warnings = %v; want no runtime and a warning on runtimes", m.GetRuntimeIDs(), m.Warnings)
	}
}

func TestAPluginThatDeclaresARuntimeHasABackendToRunIt(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","dependencies":[{"id":"sample-runtime","kind":"runtime"}]}`))
	if !errors.Is(err, manifest.ErrNoBackend) {
		t.Errorf("error = %v, want ErrNoBackend", err)
	}
}
