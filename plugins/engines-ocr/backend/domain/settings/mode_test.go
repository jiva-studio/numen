package settings

import "testing"

func TestParseRecognizeModeReadsEachModeOfTheSetting(t *testing.T) {
	for _, want := range []RecognizeMode{ModeOff, ModeWhenNoTextLayer, ModeAlways} {
		if got := ParseRecognizeMode(map[string]string{"recognizeMode": string(want)}); got != want {
			t.Errorf("recognizeMode %q read as %q", want, got)
		}
	}
}

func TestParseRecognizeModeReadsOffWhenTheValueIsAbsentOrUnknown(t *testing.T) {
	cases := map[string]map[string]string{
		"no settings":      nil,
		"no mode":          {"dpi": "300"},
		"an empty mode":    {"recognizeMode": ""},
		"an unknown mode":  {"recognizeMode": "sometimes"},
		"a mode in a case": {"recognizeMode": "ALWAYS"},
	}
	for name, values := range cases {
		if got := ParseRecognizeMode(values); got != ModeOff {
			t.Errorf("%s: read as %q, want %q", name, got, ModeOff)
		}
	}
}

func TestParseRecognizeModeIgnoresTheSettingOfBefore(t *testing.T) {
	if got := ParseRecognizeMode(map[string]string{"recognizeAutomatically": "true"}); got != ModeOff {
		t.Errorf("a stored recognizeAutomatically read as %q, want %q", got, ModeOff)
	}
}
