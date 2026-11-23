package settings

import "testing"

func TestParseRecognizerSettingsIgnoresAResolutionThatIsNotPositive(t *testing.T) {
	current := NewDefaultConfig().Recognize
	for _, dpi := range []string{"-100", "0", "not-a-number", ""} {
		if got := ParseRecognizerSettings(current, map[string]string{"dpi": dpi}); got.DPI != current.DPI {
			t.Errorf("a resolution of %q set it to %d", dpi, got.DPI)
		}
	}
}

func TestParseRecognizerSettingsReadsAPositiveResolution(t *testing.T) {
	if got := ParseRecognizerSettings(NewDefaultConfig().Recognize, map[string]string{"dpi": "1"}); got.DPI != 1 {
		t.Errorf("resolution %d, want 1", got.DPI)
	}
	if got := ParseRecognizerSettings(NewDefaultConfig().Recognize, map[string]string{"dpi": "450"}); got.DPI != 450 {
		t.Errorf("resolution %d, want 450", got.DPI)
	}
}

func TestParseRecognizerSettingsAndFindRecognizerModels(t *testing.T) {
	if addr, ok := FindRecognizer(RecognizerCustom); !ok || addr != "" {
		t.Fatalf("FindRecognizer(custom) = (%q, %v), want (\"\", true)", addr, ok)
	}
	if addr, ok := FindRecognizer("pp-ocrv6-small"); !ok || addr == "" {
		t.Fatalf("FindRecognizer(pp-ocrv6-small) = (%q, %v), want non-empty address", addr, ok)
	}
	if addr, ok := FindRecognizer("unknown-model"); ok || addr != "" {
		t.Fatalf("FindRecognizer(unknown-model) = (%q, %v), want (\"\", false)", addr, ok)
	}

	gotCustom := ParseRecognizerSettings(NewDefaultConfig().Recognize, map[string]string{
		"model":           RecognizerCustom,
		"customModelPath": "/tmp/model.onnx",
		"customDictPath":  "/tmp/dict.txt",
	})
	if gotCustom.Name != RecognizerCustom || gotCustom.Path != "/tmp/model.onnx" || gotCustom.Dict != "/tmp/dict.txt" {
		t.Fatalf("ParseRecognizerSettings(custom) = %+v", gotCustom)
	}

	gotPinned := ParseRecognizerSettings(NewDefaultConfig().Recognize, map[string]string{"model": "pp-ocrv5-latin"})
	if gotPinned.Name == "" || gotPinned.Name == RecognizerCustom || gotPinned.Name == NewDefaultConfig().Recognize.Name {
		t.Fatalf("ParseRecognizerSettings(pp-ocrv5-latin) = %+v", gotPinned)
	}
}
