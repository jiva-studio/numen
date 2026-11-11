package engine

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
)

// A dictionary the settings name is read one character per line, whatever its line ends, and the
// space is put last.
func TestADictionaryOfTheSettingsIsReadAsThePaddleOCRFormat(t *testing.T) {
	dict := filepath.Join(t.TempDir(), "letters.txt")
	if err := os.WriteFile(dict, []byte("а\nб\r\nв\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	classes, written, err := alphabet("model.onnx", settings.RecogniserModel{Dict: dict, Classes: 5})
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
	for _, mismatch := range []int64{1, 7} {
		gotClasses, _, err := alphabet("model.onnx", settings.RecogniserModel{Dict: dict, Classes: mismatch})
		if err == nil || gotClasses != 0 {
			t.Errorf("classes=%d returned (%d, %v), want (0, error)", mismatch, gotClasses, err)
		}
	}
}

func appendTag(dst []byte, field int, wire byte) []byte {
	return binary.AppendUvarint(dst, uint64(field<<3)|uint64(wire))
}

func appendBytesField(dst []byte, field int, payload []byte) []byte {
	dst = appendTag(dst, field, 2)
	dst = binary.AppendUvarint(dst, uint64(len(payload)))
	return append(dst, payload...)
}

func TestAlphabetReadsMetadataAndHandlesErrors(t *testing.T) {
	dir := t.TempDir()
	missingDict := filepath.Join(dir, "missing-dict.txt")
	if gotClasses, _, err := alphabet("model.onnx", settings.RecogniserModel{Dict: missingDict}); err == nil || gotClasses != 0 {
		t.Fatalf("missing dict returned (%d, %v), want error", gotClasses, err)
	}

	var entry []byte
	entry = appendTag(entry, 0, 0)
	entry = binary.AppendUvarint(entry, 1)
	entry = appendBytesField(entry, keyField, []byte(alphabetKey))
	entry = appendBytesField(entry, valueField, []byte("x\ny\n"))
	entry = appendBytesField(entry, 3, []byte("ignored"))

	var modelBytes []byte
	modelBytes = appendTag(modelBytes, 13, 0)
	modelBytes = binary.AppendUvarint(modelBytes, 42)
	modelBytes = appendTag(modelBytes, 15, 1)
	modelBytes = append(modelBytes, make([]byte, 8)...)
	modelBytes = appendTag(modelBytes, 3, 5)
	modelBytes = append(modelBytes, make([]byte, 4)...)
	modelBytes = appendBytesField(modelBytes, metadataField, entry)

	modelPath := filepath.Join(dir, "model.onnx")
	if err := os.WriteFile(modelPath, modelBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	classes, written, err := alphabet(modelPath, settings.RecogniserModel{})
	if err != nil || classes != 4 {
		t.Fatalf("alphabet from metadata = (%d, %q, %v), want (4, ..., nil)", classes, written, err)
	}

	for _, bad := range [][]byte{
		append(appendTag(nil, 1, 1), 1, 2, 3),
		append(appendTag(nil, 1, 5), 1, 2),
		append(appendTag(nil, 1, 2), 10, 1, 2),
		appendTag(nil, 1, 3),
	} {
		badPath := filepath.Join(dir, "bad.onnx")
		if err := os.WriteFile(badPath, bad, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readMetadata(badPath, alphabetKey); err == nil {
			t.Fatalf("readMetadata(%x) succeeded, want error", bad)
		}
	}

	t.Setenv("TMPDIR", filepath.Join(dir, "no-such-tmp"))
	if gotClasses, _, err := alphabet(modelPath, settings.RecogniserModel{}); err == nil || gotClasses != 0 {
		t.Fatalf("alphabet with invalid TMPDIR = (%d, %v), want error", gotClasses, err)
	}
}
