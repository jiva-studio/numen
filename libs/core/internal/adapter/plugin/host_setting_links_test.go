package plugin

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// newLinkedHost is a host with one plugin whose model chooses its weights, and whose path settings
// take a dictionary file and a folder.
func newLinkedHost(t *testing.T) *Host {
	t.Helper()
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "sample")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "md.numen.sample", "name": "Sample", "version": "1.0.0", "targets": ["desktop.editor"],
		"dependencies": [{"id": "small-weights", "kind": "model"}, {"id": "large-weights", "kind": "model"}],
		"settings": [
			{"key": "model", "type": "enum", "title": "Model", "default": "large", "options": [
				{"value": "large", "dependencies": ["large-weights"]},
				{"value": "small", "dependencies": ["small-weights"]}
			]},
			{"key": "url", "type": "string", "title": "URL", "default": "https://example.org", "when": {"setting": "model", "is": ["small"]}},
			{"key": "dictionary", "type": "file", "title": "Dictionary", "accept": [".txt"]},
			{"key": "cache", "type": "folder", "title": "Cache"}
		]
	}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}
	return host
}

// The plugin is launched with the weights of the selected model and receives
// the settings whose condition holds.
func TestRequirementsFollowTheModelSetting(t *testing.T) {
	host := newLinkedHost(t)
	getIDs := func() []string {
		host.mu.RLock()
		defer host.mu.RUnlock()
		return host.getRequirementIDsLocked("md.numen.sample")
	}
	if got := getIDs(); !slices.Equal(got, []string{"large-weights"}) {
		t.Errorf("requirements = %v, want the large weights", got)
	}
	if _, ok := host.GetPluginSettings("md.numen.sample")["url"]; ok {
		t.Error("the url is handed over while its condition fails")
	}

	if err := host.UpdatePluginSetting(t.Context(), "md.numen.sample", "model", "small"); err != nil {
		t.Fatal(err)
	}
	if got := getIDs(); !slices.Equal(got, []string{"small-weights"}) {
		t.Errorf("requirements = %v, want the small weights", got)
	}
	if got := host.GetPluginSettings("md.numen.sample")["url"]; got != "https://example.org" {
		t.Errorf("url = %v, want its default", got)
	}
}

// A path setting takes nothing, or a path of the kind it holds with an extension it accepts.
func TestCheckPluginSettingAcceptsPathsOfItsType(t *testing.T) {
	host := newLinkedHost(t)
	tmp := t.TempDir()
	text := filepath.Join(tmp, "letters.txt")
	model := filepath.Join(tmp, "model.onnx")
	for _, name := range []string{text, model} {
		if err := os.WriteFile(name, []byte("a\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		key   string
		value any
		want  error
	}{
		{"dictionary", "", nil},
		{"dictionary", text, nil},
		{"dictionary", model, errNotAccepted},
		{"dictionary", tmp, errNotAFile},
		{"dictionary", filepath.Join(tmp, "absent.txt"), errPathNotFound},
		{"dictionary", 3.0, errPathNotFound},
		{"cache", tmp, nil},
		{"cache", text, errNotAFolder},
		{"model", "anything", nil},
	} {
		err := host.CheckPluginSetting("md.numen.sample", tc.key, tc.value)
		if !errors.Is(err, tc.want) || (tc.want == nil && err != nil) {
			t.Errorf("%s = %v: err = %v, want %v", tc.key, tc.value, err, tc.want)
		}
	}
}
