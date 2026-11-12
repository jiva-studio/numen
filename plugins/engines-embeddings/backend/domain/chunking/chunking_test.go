package chunking

import (
	"encoding/json"
	"testing"

	embeddings "github.com/jiva-studio/numen/plugins/engines-embeddings"
)

func TestReadTokensKeepsTheSettingToTheBounds(t *testing.T) {
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
			if got := ReadTokens(tc.values); got != tc.want {
				t.Errorf("ReadTokens(%v) = %d, want %d", tc.values, got, tc.want)
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
		if one.Key != TokensKey {
			continue
		}
		if one.Default != any(float64(DefaultTokens)) || one.Min == nil || one.Max == nil ||
			int(*one.Min) != MinTokens || int(*one.Max) != MaxTokens {
			t.Errorf("manifest setting = %+v, want default/min/max %d/%d/%d", one, DefaultTokens, MinTokens, MaxTokens)
		}
		return
	}
	t.Errorf("the manifest declares no %s setting", TokensKey)
}

func TestClampTokensLimitsTheSizeToTheModelLimit(t *testing.T) {
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
			if got := ClampTokens(tc.requested, tc.limit); got != tc.want {
				t.Errorf("ClampTokens(%d, %d) = %d, want %d", tc.requested, tc.limit, got, tc.want)
			}
		})
	}
}

func TestReadMinLengthKeepsTheSettingToTheBounds(t *testing.T) {
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
			if got := ReadMinLength(tc.values); got != tc.want {
				t.Errorf("ReadMinLength(%v) = %d, want %d", tc.values, got, tc.want)
			}
		})
	}
}

func TestMinLengthIsReadFromTheCamelCaseKey(t *testing.T) {
	if got := ReadMinLength(map[string]string{"minChunkLength": "120"}); got != 120 {
		t.Errorf("ReadMinLength = %d, want 120", got)
	}
}

func TestMinLengthIsNotReadFromTheSnakeCaseKey(t *testing.T) {
	if got := ReadMinLength(map[string]string{"min_chunk_length": "120"}); got != DefaultMinLength {
		t.Errorf("ReadMinLength = %d, want the default %d", got, DefaultMinLength)
	}
}
