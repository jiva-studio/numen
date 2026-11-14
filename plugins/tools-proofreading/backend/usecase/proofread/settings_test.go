package proofread

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRequestSizesAreReadFromTheCamelCaseKeys(t *testing.T) {
	got := ParseConfig(map[string]string{
		"batchSize": "7", "inFlight": "3", "maxEditDistance": "0.45",
	}, DefaultConfig())

	if got.BatchSize != 7 || got.InFlight != 3 || got.MaxEditDistance != 0.45 {
		t.Errorf("batch size %d, in flight %d, max edit distance %v; want 7, 3 and 0.45",
			got.BatchSize, got.InFlight, got.MaxEditDistance)
	}
}

func TestRequestSizesAreNotReadFromTheSnakeCaseKeys(t *testing.T) {
	before := DefaultConfig()

	got := ParseConfig(map[string]string{
		"batch_size": "7", "in_flight": "3", "max_edit_distance": "0.45",
	}, before)

	if got != before {
		t.Errorf("config = %+v, want it unchanged at %+v", got, before)
	}
}

func TestLanguageAndPromptAreReadFromTheirKeys(t *testing.T) {
	got := ParseConfig(map[string]string{"language": "Hindi", "prompt": "Keep the old spelling of names."}, DefaultConfig())

	if got.Language != "Hindi" || got.Prompt != "Keep the old spelling of names." {
		t.Errorf("language %q, prompt %q", got.Language, got.Prompt)
	}
}

func TestAnEmptyLanguageAndPromptClearTheOnesSet(t *testing.T) {
	base := DefaultConfig()
	base.Language, base.Prompt = "Hindi", "Keep names."

	got := ParseConfig(map[string]string{"language": "", "prompt": ""}, base)

	if got.Language != "" || got.Prompt != "" {
		t.Errorf("language %q, prompt %q, want both cleared", got.Language, got.Prompt)
	}
}

func TestSettingsWithoutLanguageAndPromptKeepTheOnesSet(t *testing.T) {
	base := DefaultConfig()
	base.Language, base.Prompt = "Hindi", "Keep names."

	got := ParseConfig(map[string]string{"batchSize": "9"}, base)

	if got.Language != "Hindi" || got.Prompt != "Keep names." || got.BatchSize != 9 {
		t.Errorf("config = %+v, want language and prompt kept and batch size 9", got)
	}
}

func TestLanguageAndPromptLeaveTheSizesAsTheyAre(t *testing.T) {
	before := DefaultConfig()

	got := ParseConfig(map[string]string{"language": "Hindi", "prompt": "Keep names."}, before)

	got.Language, got.Prompt = "", ""
	if got != before {
		t.Errorf("sizes = %+v, want %+v", got, before)
	}
}

func TestTheLanguageIsOneLineWithoutSurroundingSpace(t *testing.T) {
	got := ParseConfig(map[string]string{"language": "  Church\n \t Slavonic\r\n "}, DefaultConfig())

	if got.Language != "Church Slavonic" {
		t.Errorf("language = %q, want %q", got.Language, "Church Slavonic")
	}
}

func TestALongLanguageIsCutAtTheLimitByRunes(t *testing.T) {
	long := strings.Repeat("я", MaxLanguageLen+30)

	got := ParseConfig(map[string]string{"language": long}, DefaultConfig())

	if want := strings.Repeat("я", MaxLanguageLen); got.Language != want {
		t.Errorf("language has %d runes, want %d", utf8.RuneCountInString(got.Language), MaxLanguageLen)
	}
}

func TestThePromptIsTrimmedAndKeepsItsLineBreaks(t *testing.T) {
	got := ParseConfig(map[string]string{"prompt": "\n  Keep names.\n\nKeep dates.  \n"}, DefaultConfig())

	if want := "Keep names.\n\nKeep dates."; got.Prompt != want {
		t.Errorf("prompt = %q, want %q", got.Prompt, want)
	}
}

func TestALongPromptIsCutAtTheLimitByRunes(t *testing.T) {
	long := strings.Repeat("ś", MaxPromptLen+50)

	got := ParseConfig(map[string]string{"prompt": long}, DefaultConfig())

	if want := strings.Repeat("ś", MaxPromptLen); got.Prompt != want {
		t.Errorf("prompt has %d runes, want %d", utf8.RuneCountInString(got.Prompt), MaxPromptLen)
	}
}
