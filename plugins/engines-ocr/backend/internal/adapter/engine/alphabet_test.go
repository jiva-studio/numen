package engine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
	classes, written, err := loadAlphabet("model.onnx", settings.RecognizerModel{Dict: dict, Classes: 5})
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
		gotClasses, _, err := loadAlphabet("model.onnx", settings.RecognizerModel{Dict: dict, Classes: mismatch})
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
	if gotClasses, _, err := loadAlphabet("model.onnx", settings.RecognizerModel{Dict: missingDict}); !errors.Is(err, os.ErrNotExist) || gotClasses != 0 {
		t.Fatalf("missing dict returned (%d, %v), want os.ErrNotExist", gotClasses, err)
	}
	if gotClasses, _, err := loadAlphabet(filepath.Join(dir, "missing-model.onnx"), settings.RecognizerModel{}); !errors.Is(err, os.ErrNotExist) || gotClasses != 0 {
		t.Fatalf("missing model returned (%d, %v), want os.ErrNotExist", gotClasses, err)
	}

	var wrong13 []byte
	wrong13 = appendBytesField(wrong13, 1, []byte(alphabetKey))
	wrong13 = appendBytesField(wrong13, 2, []byte("wrong13\n"))

	var wrong15 []byte
	wrong15 = appendBytesField(wrong15, 1, []byte(alphabetKey))
	wrong15 = appendBytesField(wrong15, 2, []byte("wrong15\n"))

	var noKeyEntry []byte
	noKeyEntry = appendBytesField(noKeyEntry, 2, []byte("no-key\n"))

	var entry []byte
	entry = appendTag(entry, 0, 0)
	entry = binary.AppendUvarint(entry, 1)
	entry = appendBytesField(entry, 1, []byte(alphabetKey))
	entry = appendBytesField(entry, 2, []byte("x\ny\n"))
	entry = appendBytesField(entry, 3, []byte("ignored"))

	var modelBytes []byte
	modelBytes = appendBytesField(modelBytes, 13, wrong13)
	modelBytes = appendBytesField(modelBytes, 15, wrong15)
	modelBytes = appendTag(modelBytes, 1, 1)
	modelBytes = append(modelBytes, bytes.Repeat([]byte{0xff}, 8)...)
	modelBytes = appendTag(modelBytes, 3, 5)
	modelBytes = append(modelBytes, bytes.Repeat([]byte{0xff}, 4)...)
	modelBytes = appendBytesField(modelBytes, 14, noKeyEntry)
	modelBytes = appendBytesField(modelBytes, 14, entry)

	modelPath := filepath.Join(dir, "model.onnx")
	if err := os.WriteFile(modelPath, modelBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	classes, written, err := loadAlphabet(modelPath, settings.RecognizerModel{})
	if err != nil || classes != 4 {
		t.Fatalf("alphabet from metadata = (%d, %q, %v), want (4, ..., nil)", classes, written, err)
	}

	if _, err := readMetadata(modelPath, "absent-key"); err == nil || !strings.Contains(err.Error(), `no "absent-key" in the model's metadata`) {
		t.Fatalf("readMetadata(absent-key) = %v, want missing key error", err)
	}

	for _, bad := range [][]byte{
		{0x80},
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
	var pathErr *fs.PathError
	if gotClasses, _, err := loadAlphabet(modelPath, settings.RecognizerModel{}); !errors.As(err, &pathErr) || gotClasses != 0 {
		t.Fatalf("alphabet with invalid TMPDIR = (%d, %v), want *fs.PathError", gotClasses, err)
	}
}

func TestParseTagReadPayloadAndParseEntryBoundaries(t *testing.T) {
	overflowVarint := append(bytes.Repeat([]byte{0x80}, 10), 0x02)

	for _, raw := range [][]byte{nil, {0x80}, overflowVarint} {
		if field, wire, n, ok := parseTag(raw); ok || field != 0 || wire != 0 || n != 0 {
			t.Fatalf("parseTag(%x) = (%d, %d, %d, %v), want (0, 0, 0, false)", raw, field, wire, n, ok)
		}
	}
	if field, wire, n, ok := parseTag([]byte{(14 << 3) | 5}); !ok || field != 14 || wire != 5 || n != 1 {
		t.Fatalf("parseTag(14, 5) = (%d, %d, %d, %v), want (14, 5, 1, true)", field, wire, n, ok)
	}

	if payload, n, ok := readPayload(nil, 0); ok || payload != nil || n != 0 {
		t.Fatalf("readPayload(nil, 0) = (%x, %d, %v)", payload, n, ok)
	}
	if payload, n, ok := readPayload(overflowVarint, 0); ok || payload != nil || n != 0 {
		t.Fatalf("readPayload(overflow, 0) = (%x, %d, %v)", payload, n, ok)
	}
	if payload, n, ok := readPayload([]byte{0x05}, 0); !ok || payload != nil || n != 1 {
		t.Fatalf("readPayload(0x05, 0) = (%x, %d, %v), want (nil, 1, true)", payload, n, ok)
	}

	if payload, n, ok := readPayload(make([]byte, 7), 1); ok || payload != nil || n != 0 {
		t.Fatalf("readPayload(7 bytes, 1) = (%x, %d, %v)", payload, n, ok)
	}
	if payload, n, ok := readPayload(make([]byte, 8), 1); !ok || payload != nil || n != 8 {
		t.Fatalf("readPayload(8 bytes, 1) = (%x, %d, %v), want (nil, 8, true)", payload, n, ok)
	}

	if payload, n, ok := readPayload(nil, 2); ok || payload != nil || n != 0 {
		t.Fatalf("readPayload(nil, 2) = (%x, %d, %v)", payload, n, ok)
	}
	if payload, n, ok := readPayload(overflowVarint, 2); ok || payload != nil || n != 0 {
		t.Fatalf("readPayload(overflow, 2) = (%x, %d, %v)", payload, n, ok)
	}
	if payload, n, ok := readPayload([]byte{2, 'a'}, 2); ok || payload != nil || n != 0 {
		t.Fatalf("readPayload(truncated, 2) = (%x, %d, %v)", payload, n, ok)
	}
	if payload, n, ok := readPayload([]byte{3, 'a', 'b', 'c'}, 2); !ok || string(payload) != "abc" || n != 4 {
		t.Fatalf("readPayload(abc, 2) = (%q, %d, %v), want (\"abc\", 4, true)", payload, n, ok)
	}

	if payload, n, ok := readPayload(make([]byte, 3), 5); ok || payload != nil || n != 0 {
		t.Fatalf("readPayload(3 bytes, 5) = (%x, %d, %v)", payload, n, ok)
	}
	if payload, n, ok := readPayload(make([]byte, 4), 5); !ok || payload != nil || n != 4 {
		t.Fatalf("readPayload(4 bytes, 5) = (%x, %d, %v), want (nil, 4, true)", payload, n, ok)
	}

	if payload, n, ok := readPayload(make([]byte, 8), 3); ok || payload != nil || n != 0 {
		t.Fatalf("readPayload(wire 3) = (%x, %d, %v)", payload, n, ok)
	}

	if key, val, ok := parseEntry([]byte{0x80}); ok || key != "" || val != "" {
		t.Fatalf("parseEntry(bad tag) = (%q, %q, %v)", key, val, ok)
	}
	if key, val, ok := parseEntry(append(appendTag(nil, 1, 2), 5, 'a')); ok || key != "" || val != "" {
		t.Fatalf("parseEntry(bad payload) = (%q, %q, %v)", key, val, ok)
	}
	if key, val, ok := parseEntry(appendBytesField(nil, 2, []byte("only-value"))); ok || key != "" || val != "only-value" {
		t.Fatalf("parseEntry(no key) = (%q, %q, %v), want (\"\", \"only-value\", false)", key, val, ok)
	}
}
