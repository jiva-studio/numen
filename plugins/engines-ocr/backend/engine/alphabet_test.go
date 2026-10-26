package engine

import (
	"os"
	"path/filepath"
	"testing"
)

// A dictionary the settings name is read one character per line, whatever its line ends, and the
// space is put last.
func TestADictionaryOfTheSettingsIsReadAsThePaddleOCRFormat(t *testing.T) {
	dict := filepath.Join(t.TempDir(), "letters.txt")
	if err := os.WriteFile(dict, []byte("а\nб\r\nв\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	classes, written, err := alphabet("model.onnx", RecogniserModel{Dict: dict})
	if err != nil {
		t.Fatal(err)
	}
	if classes != 5 {
		t.Errorf("classes = %d, want the three letters, the space and the blank", classes)
	}
	raw, err := os.ReadFile(written)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(raw); got != "а\nб\nв\n \n" {
		t.Errorf("the alphabet written is %q", got)
	}
	if _, _, err := alphabet("model.onnx", RecogniserModel{Dict: dict, Classes: 7}); err == nil {
		t.Error("a dictionary of three letters is taken for a model of five")
	}
}
