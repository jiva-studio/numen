package manifestfile

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

func TestManifestTreatsAnONNXRuntimeDependencyAsTheRuntimesEntry(t *testing.T) {
	legacy, err := ReadManifest(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","runtimes":["onnxruntime"],"backend":{"entrypoint":"bin/a"}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	explicit, err := ReadManifest(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","dependencies":[{"id":"onnxruntime","kind":"runtime"}]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(legacy.GetRuntimeIDs(), explicit.GetRuntimeIDs()) {
		t.Errorf("runtime ids = %v, want %v", explicit.GetRuntimeIDs(), legacy.GetRuntimeIDs())
	}
	same := func(a, b pluginport.DependencyRequirement) bool {
		return a.ID == b.ID && a.Kind == b.Kind && a.SourceURL == b.SourceURL && a.PluginDir == b.PluginDir
	}
	if !slices.EqualFunc(legacy.GetDependencyRequirements("/p", nil), explicit.GetDependencyRequirements("/p", nil), same) {
		t.Errorf("requirements = %v, want %v", explicit.GetDependencyRequirements("/p", nil), legacy.GetDependencyRequirements("/p", nil))
	}
}
