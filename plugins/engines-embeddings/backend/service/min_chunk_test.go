package service

import "testing"

func TestReadMinChunkLength(t *testing.T) {
	cases := []struct {
		name   string
		values map[string]string
		want   int
	}{
		{name: "absent gives the default", values: map[string]string{}, want: 40},
		{name: "empty gives the default", values: map[string]string{"minChunkLength": ""}, want: 40},
		{name: "text gives the default", values: map[string]string{"minChunkLength": "many"}, want: 40},
		{name: "NaN gives the default", values: map[string]string{"minChunkLength": "NaN"}, want: 40},
		{name: "zero embeds everything", values: map[string]string{"minChunkLength": "0"}, want: 0},
		{name: "a whole number", values: map[string]string{"minChunkLength": "120"}, want: 120},
		{name: "spaces around", values: map[string]string{"minChunkLength": " 7 "}, want: 7},
		{name: "a fraction is cut", values: map[string]string{"minChunkLength": "12.9"}, want: 12},
		{name: "below zero is zero", values: map[string]string{"minChunkLength": "-5"}, want: 0},
		{name: "the maximum", values: map[string]string{"minChunkLength": "1000"}, want: 1000},
		{name: "above the maximum is the maximum", values: map[string]string{"minChunkLength": "5000"}, want: 1000},
		{name: "infinity is the maximum", values: map[string]string{"minChunkLength": "+Inf"}, want: 1000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := readMinChunkLength(tc.values); got != tc.want {
				t.Errorf("readMinChunkLength(%v) = %d, want %d", tc.values, got, tc.want)
			}
		})
	}
}

// The model info carries the setting, the default before the host configures the plugin, and the
// value the host sends after.
func TestGetModelInfoReportsMinChunkLength(t *testing.T) {
	srv := newEmbeddingServer()
	info, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if info.GetMinChunkLength() != 40 {
		t.Errorf("before configure = %d, want 40", info.GetMinChunkLength())
	}
	for value, want := range map[string]int32{"0": 0, "250": 250, "9999": 1000} {
		srv.updateConfig(map[string]string{"source": "local", "minChunkLength": value})
		info, err = srv.GetModelInfo(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if info.GetMinChunkLength() != want {
			t.Errorf("configured %s: reported %d, want %d", value, info.GetMinChunkLength(), want)
		}
	}
}

// Changing the setting does not change the model that makes the vectors.
func TestMinChunkLengthIsNotPartOfTheModel(t *testing.T) {
	srv := newEmbeddingServer()
	srv.updateConfig(map[string]string{"source": "local", "minChunkLength": "40"})
	before, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	srv.updateConfig(map[string]string{"source": "local", "minChunkLength": "0"})
	after, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if before.Name != after.Name || before.Dimensions != after.Dimensions || before.MaxTokens != after.MaxTokens ||
		before.Pooling != after.Pooling || before.From != after.From {
		t.Errorf("model changed with the setting: %+v -> %+v", before, after)
	}
}
