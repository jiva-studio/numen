package reasoning

import "testing"

func TestOpenRouterTakesEveryLevelForEveryModel(t *testing.T) {
	levels := GetLevels("openrouter", "any/model")

	if len(levels) != 3 || levels[0].Value != "low" || levels[2].Title != "High" {
		t.Errorf("levels = %+v", levels)
	}
}

func TestOpenAITakesLevelsForItsReasoningModelsOnly(t *testing.T) {
	for _, model := range []string{"o3-mini", "O1", "gpt-5", " o4-mini "} {
		if len(GetLevels("openai", model)) != 3 {
			t.Errorf("%q takes no levels", model)
		}
	}
	if levels := GetLevels("openai", "gpt-4o"); len(levels) != 0 {
		t.Errorf("gpt-4o takes %+v", levels)
	}
}

func TestACompatibleServerTakesNoLevel(t *testing.T) {
	if levels := GetLevels("openai-compatible", "o3"); len(levels) != 0 {
		t.Errorf("levels = %+v", levels)
	}
}

func TestALevelTheModelDoesNotTakeIsNotSupported(t *testing.T) {
	if !IsSupported("openrouter", "m", "high") || !IsSupported("openai", "gpt-4o", "") {
		t.Error("a level the model takes, or none, was refused")
	}
	if IsSupported("openai", "gpt-4o", "high") || IsSupported("openrouter", "m", "extreme") {
		t.Error("a level the model does not take was accepted")
	}
}
