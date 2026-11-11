package settings

import "testing"

func TestParseRecognizerSettingsIgnoresAResolutionThatIsNotPositive(t *testing.T) {
	current := Defaults().Recognize
	for _, dpi := range []string{"-100", "0", "not-a-number", ""} {
		if got := ParseRecognizerSettings(current, map[string]string{"dpi": dpi}); got.DPI != current.DPI {
			t.Errorf("a resolution of %q set it to %d", dpi, got.DPI)
		}
	}
}

func TestParseRecognizerSettingsReadsAPositiveResolution(t *testing.T) {
	if got := ParseRecognizerSettings(Defaults().Recognize, map[string]string{"dpi": "450"}); got.DPI != 450 {
		t.Errorf("resolution %d, want 450", got.DPI)
	}
}
