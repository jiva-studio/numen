package serving

import "testing"

// The model info carries the setting, the default before the host configures the plugin, and the
// value the host sends after.
func TestGetModelInfoReportsMinChunkLength(t *testing.T) {
	srv := newEmbeddingServer()
	info, err := srv.GetModelInfo()
	if err != nil {
		t.Fatal(err)
	}
	if info.MinChunkLength != 40 {
		t.Errorf("before configure = %d, want 40", info.MinChunkLength)
	}
	for value, want := range map[string]int{"0": 0, "250": 250, "9999": 1000} {
		srv.updateConfig(map[string]string{"provider": "local", "minChunkLength": value}, nil)
		info, err = srv.GetModelInfo()
		if err != nil {
			t.Fatal(err)
		}
		if info.MinChunkLength != want {
			t.Errorf("configured %s: reported %d, want %d", value, info.MinChunkLength, want)
		}
	}
}

// Changing the setting does not change the model that makes the vectors.
func TestMinChunkLengthIsNotPartOfTheModel(t *testing.T) {
	srv := newEmbeddingServer()
	srv.updateConfig(map[string]string{"provider": "local", "minChunkLength": "40"}, nil)
	before, err := srv.GetModelInfo()
	if err != nil {
		t.Fatal(err)
	}
	srv.updateConfig(map[string]string{"provider": "local", "minChunkLength": "0"}, nil)
	after, err := srv.GetModelInfo()
	if err != nil {
		t.Fatal(err)
	}
	if before.Name != after.Name || before.Dimensions != after.Dimensions || before.MaxTokens != after.MaxTokens ||
		before.Pooling != after.Pooling || before.From != after.From {
		t.Errorf("model changed with the setting: %+v -> %+v", before, after)
	}
}
