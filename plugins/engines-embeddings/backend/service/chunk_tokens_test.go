package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	embeddings "github.com/jiva-studio/numen/plugins/engines-embeddings"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
)

func TestReadChunkTokensKeepsTheSettingToTheBounds(t *testing.T) {
	cases := []struct {
		name   string
		values map[string]string
		want   int
	}{
		{name: "an absent key gives the default", values: map[string]string{}, want: 256},
		{name: "a value that is no number gives the default", values: map[string]string{"chunkTokens": "abc"}, want: 256},
		{name: "an empty value gives the default", values: map[string]string{"chunkTokens": ""}, want: 256},
		{name: "zero is brought up to the minimum", values: map[string]string{"chunkTokens": "0"}, want: 16},
		{name: "below the minimum is the minimum", values: map[string]string{"chunkTokens": "15"}, want: 16},
		{name: "the minimum", values: map[string]string{"chunkTokens": "16"}, want: 16},
		{name: "a negative number is the minimum", values: map[string]string{"chunkTokens": "-40"}, want: 16},
		{name: "a fraction is cut to its whole part", values: map[string]string{"chunkTokens": "256.9"}, want: 256},
		{name: "spaces around a number are ignored", values: map[string]string{"chunkTokens": " 300 "}, want: 300},
		{name: "the maximum", values: map[string]string{"chunkTokens": "8192"}, want: 8192},
		{name: "above the maximum is the maximum", values: map[string]string{"chunkTokens": "100000"}, want: 8192},
		{name: "infinity is the maximum", values: map[string]string{"chunkTokens": "+Inf"}, want: 8192},
		{name: "NaN gives the default", values: map[string]string{"chunkTokens": "NaN"}, want: 256},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ReadChunkTokens(tc.values); got != tc.want {
				t.Errorf("ReadChunkTokens(%v) = %d, want %d", tc.values, got, tc.want)
			}
		})
	}
}

func TestChunkTokensBoundsAreTheBoundsOfTheManifestSetting(t *testing.T) {
	var manifest struct {
		Settings []struct {
			Key     string   `json:"key"`
			Default any      `json:"default"`
			Min     *float64 `json:"min"`
			Max     *float64 `json:"max"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(embeddings.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, one := range manifest.Settings {
		if one.Key != settingChunkTokens {
			continue
		}
		if one.Default != any(float64(defaultChunkTokens)) || one.Min == nil || one.Max == nil ||
			int(*one.Min) != minChunkTokens || int(*one.Max) != maxChunkTokens {
			t.Errorf("manifest setting = %+v, want default/min/max %d/%d/%d", one, defaultChunkTokens, minChunkTokens, maxChunkTokens)
		}
		return
	}
	t.Errorf("the manifest declares no %s setting", settingChunkTokens)
}

func TestClampChunkTokensLimitsTheSizeToTheModelLimit(t *testing.T) {
	cases := []struct {
		name             string
		requested, limit int
		want             int
	}{
		{name: "a size under the limit stays", requested: 400, limit: 512, want: 400},
		{name: "a size at the limit stays", requested: 512, limit: 512, want: 512},
		{name: "a size over the limit is the limit", requested: 1000, limit: 512, want: 512},
		{name: "a model that names no limit holds nothing", requested: 1000, limit: 0, want: 1000},
		{name: "a negative limit holds nothing", requested: 1000, limit: -1, want: 1000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClampChunkTokens(tc.requested, tc.limit); got != tc.want {
				t.Errorf("ClampChunkTokens(%d, %d) = %d, want %d", tc.requested, tc.limit, got, tc.want)
			}
		})
	}
}

func getReportedChunkTokens(t *testing.T, srv *embeddingServer) int32 {
	t.Helper()
	info, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return info.GetChunkTokens()
}

func TestGetModelInfoReportsTheDefaultChunkTokensOfALocalModelOf512(t *testing.T) {
	srv := newEmbeddingServer()
	srv.updateConfig(map[string]string{"provider": "local", "model": engine.DefaultModel})
	info, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if info.GetMaxTokens() != 512 || info.GetChunkTokens() != 256 {
		t.Errorf("max tokens %d, chunk tokens %d; want 512 and 256", info.GetMaxTokens(), info.GetChunkTokens())
	}
}

func TestGetModelInfoReportsTheDefaultChunkTokensBeforeTheHostConfiguresThePlugin(t *testing.T) {
	if got := getReportedChunkTokens(t, newEmbeddingServer()); got != 256 {
		t.Errorf("chunk tokens before configure = %d, want 256", got)
	}
}

func TestGetModelInfoNeverReportsMoreChunkTokensThanTheModelReads(t *testing.T) {
	srv := newEmbeddingServer()
	srv.updateConfig(map[string]string{"provider": "local", "model": engine.DefaultModel, "chunkTokens": "1000"})
	if got := getReportedChunkTokens(t, srv); got != 512 {
		t.Errorf("chunk tokens for 1000 on a model of 512 = %d, want 512", got)
	}
}

func TestGetModelInfoReportsTheRequestedChunkTokensWithinTheBounds(t *testing.T) {
	srv := newEmbeddingServer()
	for value, want := range map[string]int32{"abc": 256, "0": 16, "15": 16, "300": 300, "256.9": 256} {
		srv.updateConfig(map[string]string{"provider": "local", "model": engine.DefaultModel, "chunkTokens": value})
		if got := getReportedChunkTokens(t, srv); got != want {
			t.Errorf("configured %q: reported %d, want %d", value, got, want)
		}
	}
}

func TestAModelOfALowerLimitLowersChunkTokensAndTheOldModelRestoresThem(t *testing.T) {
	srv := newEmbeddingServer()
	dir := t.TempDir()
	srv.profilesDir = dir
	values := map[string]string{"provider": "local", "model": engine.DefaultModel, "chunkTokens": "400"}

	srv.updateConfig(values)
	first := getReportedChunkTokens(t, srv)
	writeFile(t, filepath.Join(dir, "small.json"), `{"id":"Xenova/multilingual-e5-small","maxTokens":256}`)
	srv.updateConfig(map[string]string{"provider": "local", "model": engine.DefaultModel})
	lowered := getReportedChunkTokens(t, srv)
	if err := os.Remove(filepath.Join(dir, "small.json")); err != nil {
		t.Fatal(err)
	}
	srv.updateConfig(map[string]string{"provider": "local", "model": engine.DefaultModel})
	restored := getReportedChunkTokens(t, srv)

	if first != 400 || lowered != 256 || restored != 400 {
		t.Errorf("chunk tokens = %d, %d, %d; want 400, 256, 400", first, lowered, restored)
	}
}

func TestAChangeOfChunkTokensDoesNotChangeTheModelNorItsHandling(t *testing.T) {
	srv := newEmbeddingServer()
	srv.updateConfig(map[string]string{"provider": "local", "model": engine.DefaultModel, "chunkTokens": "256"})
	before, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	srv.updateConfig(map[string]string{"provider": "local", "model": engine.DefaultModel, "chunkTokens": "128"})
	after, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if after.GetChunkTokens() != 128 || before.GetChunkTokens() != 256 {
		t.Fatalf("chunk tokens = %d then %d, want 256 then 128", before.GetChunkTokens(), after.GetChunkTokens())
	}
	after.ChunkTokens = before.ChunkTokens
	if after != before {
		t.Errorf("model changed with the setting: %+v -> %+v", before, after)
	}
}

func TestTheMockModelReportsChunkTokensLimitedByItsMaxTokens(t *testing.T) {
	t.Setenv(envMockML, "true")
	srv := newEmbeddingServer()
	srv.updateConfig(map[string]string{"chunkTokens": "300"})
	if got := getReportedChunkTokens(t, srv); got != 300 {
		t.Errorf("mock chunk tokens = %d, want 300", got)
	}
	srv.updateConfig(map[string]string{"chunkTokens": "2000"})
	if got := getReportedChunkTokens(t, srv); got != 512 {
		t.Errorf("mock chunk tokens for 2000 = %d, want its limit 512", got)
	}
}
