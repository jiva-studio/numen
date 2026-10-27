package service

import (
	"bytes"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func getLevelNames(model AgentModel) []string {
	var names []string
	for _, one := range model.Reasoning {
		names = append(names, one.Name)
	}
	return names
}

func TestTheModelListCarriesTheLevelsEachModelSupports(t *testing.T) {
	response := `{"id":1,"result":{}}` + "\n" +
		`{"id":2,"result":{"data":[` +
		`{"model":"deep","displayName":"Deep","defaultReasoningEffort":"high",` +
		`"supportedReasoningEfforts":[{"reasoningEffort":"high"},{"reasoningEffort":"low"},{"reasoningEffort":"high"}]},` +
		`{"model":"plain","displayName":"Plain"}` +
		`]}}` + "\n"
	models, err := listCodexModels(&bytes.Buffer{}, strings.NewReader(response))
	if err != nil {
		t.Fatal(err)
	}
	if got := getLevelNames(models[0]); !slices.Equal(got, []string{"low", "high"}) || models[0].DefaultReasoning != "high" {
		t.Errorf("the levels are %v, the default %q", got, models[0].DefaultReasoning)
	}
	if len(models[1].Reasoning) != 0 || models[1].DefaultReasoning != "" {
		t.Errorf("a model listing none offers %v", models[1].Reasoning)
	}
}

func TestTheModelCacheCarriesTheLevelsEachModelSupports(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o700); err != nil {
		t.Fatal(err)
	}
	cache := `{"models":[{"slug":"deep","display_name":"Deep","default_reasoning_level":"medium",` +
		`"supported_reasoning_levels":[{"effort":"low"},{"effort":"medium"},{"effort":"xhigh"}]}]}`
	if err := os.WriteFile(filepath.Join(home, ".codex", "models_cache.json"), []byte(cache), 0o600); err != nil {
		t.Fatal(err)
	}
	models := readCodexCache()
	if got := getLevelNames(models[0]); !slices.Equal(got, []string{"low", "medium", "xhigh"}) || models[0].DefaultReasoning != "medium" {
		t.Errorf("the levels are %v, the default %q", got, models[0].DefaultReasoning)
	}
}

func TestTheLevelOfARequestIsHandedToTheProgramAsConfiguration(t *testing.T) {
	got := appendModelArguments(nil, &agentrun.AskAgentRequest{Model: "deep", Reasoning: "high"})
	want := []string{"--model", "deep", "-c", `model_reasoning_effort="high"`}
	if !slices.Equal(got, want) {
		t.Errorf("the arguments are %v, want %v", got, want)
	}
	for _, level := range []string{"", "bogus"} {
		if got := appendModelArguments(nil, &agentrun.AskAgentRequest{Reasoning: level}); len(got) != 0 {
			t.Errorf("the level %q gave %v", level, got)
		}
	}
}
