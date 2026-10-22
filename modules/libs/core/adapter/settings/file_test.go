package settings_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// writeSettingsFile makes a settings file in a folder of this test's own.
func writeSettingsFile(t *testing.T, output string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "numen.json")
	if output != "" {
		if err := os.WriteFile(path, []byte(output), 0o600); err != nil {
			t.Fatalf("writing the file: %v", err)
		}
	}
	return path
}

func TestReadPassesBackEveryByteAsItExists(t *testing.T) {
	content := "{\n  \"agent\": { \"use\": \"claude\" }\n}\n"
	raw, err := settings.Read(writeSettingsFile(t, content))
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if string(raw) != content {
		t.Errorf("read back %q, wanted %q", raw, content)
	}
}

func TestReadOfAFileThatIsNotThereIsAnEmptyObject(t *testing.T) {
	raw, err := settings.Read(filepath.Join(t.TempDir(), "numen.json"))
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if strings.TrimSpace(string(raw)) != "{}" {
		t.Errorf("read back %q, wanted an empty object", raw)
	}
}

func TestWriteKeepsTheBytesAsTheyWereTyped(t *testing.T) {
	path := writeSettingsFile(t, "{}\n")
	// Two spaces of indent, a section in an order of somebody's own, and a
	// number written to two places.
	content := "{\n  \"indexing\": {\n    \"proofreading\": { \"max_edit_distance\": 0.30 }\n  }\n}\n"

	if err := settings.Write(path, []byte(content), nil, newDocument()); err != nil {
		t.Fatalf("writing: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading it back: %v", err)
	}
	if string(raw) != content {
		t.Errorf("the file holds %q, wanted %q", raw, content)
	}
}

func TestWriteRefusesWhatIsNotJSONAndPreservesTheFileAsItWas(t *testing.T) {
	target := "{\n  \"agent\": { \"use\": \"claude\" }\n}\n"
	path := writeSettingsFile(t, target)

	err := settings.Write(
		path, []byte("{ \"agent\": { \"use\": \"claude\" } // a comment\n}"), nil, newDocument())
	if !errors.Is(err, port.ErrNotASetting) {
		t.Fatalf("failed with %v, wanted an error", err)
	}
	if !strings.Contains(err.Error(), "byte") {
		t.Errorf("said %q, wanted it to name where the trouble is", err)
	}

	raw, _ := os.ReadFile(path)
	if string(raw) != target {
		t.Errorf("the file holds %q, wanted it left as it was", raw)
	}
}

func TestWriteRefusesAValueOfTheWrongKind(t *testing.T) {
	path := writeSettingsFile(t, "{}\n")

	err := settings.Write(
		path, []byte(`{"appearance": {"interface_scale": "large"}}`), nil, newDocument())
	if !errors.Is(err, port.ErrNotASetting) {
		t.Fatalf("failed with %v, wanted an error", err)
	}
	if !strings.Contains(err.Error(), "byte") {
		t.Errorf("said %q, wanted it to name where the trouble is", err)
	}
}

// A settings file is one object. Anything else read into the settings leaves
// them at their defaults, and defaults that hold say nothing about the bytes.
func TestWriteRefusesWhatIsNotAnObject(t *testing.T) {
	for _, content := range []string{"null", "42", `"hello"`, "[1,2]", "true"} {
		target := "{\n  \"agent\": { \"use\": \"claude\" }\n}\n"
		path := writeSettingsFile(t, target)

		err := settings.Write(path, []byte(content), nil, newDocument())
		if !errors.Is(err, port.ErrNotASetting) {
			t.Errorf("%s: failed with %v, wanted an error", content, err)
		}

		raw, _ := os.ReadFile(path)
		if string(raw) != target {
			t.Errorf("%s: the file holds %q, wanted it left as it was", content, raw)
		}
	}
}

// A save refuses a name a section holds twice, so the editor refuses to write
// one: the settings are read from the last of the two and a save patches the
// first.
func TestWriteRefusesANameASectionContainsTwice(t *testing.T) {
	for what, content := range map[string]string{
		"a section":        `{"appearance":{"mode":"dark"},"appearance":{"mode":"light"}}`,
		"a field":          `{"appearance":{"mode":"dark","mode":"light"}}`,
		"a field far down": `{"indexing":{"embedding":{"indexing":{"use":"a","use":"b"}}}}`,
	} {
		target := "{\n  \"agent\": { \"use\": \"claude\" }\n}\n"
		path := writeSettingsFile(t, target)

		err := settings.Write(path, []byte(content), nil, newDocument())
		if !errors.Is(err, port.ErrNotASetting) {
			t.Errorf("%s: failed with %v, wanted an error", what, err)
		} else if !strings.Contains(err.Error(), "one of a name") {
			t.Errorf("%s: said %q, wanted it to name the trouble", what, err)
		}

		raw, _ := os.ReadFile(path)
		if string(raw) != target {
			t.Errorf("%s: the file holds %q, wanted it left as it was", what, raw)
		}
	}
}

func TestWhatIsSaidDoesNotRepeatWhatExistsInTheFile(t *testing.T) {
	path := writeSettingsFile(t, "{}\n")
	// A key is the one thing in a settings file worth keeping to itself.
	secret := "sk-not-a-real-key-0000"

	err := settings.Write(
		path, []byte(`{"appearance": {"interface_scale": "`+secret+`"}}`), nil, newDocument())
	if err == nil {
		t.Fatal("wanted an error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("got error %q repeating file content", err)
	}
}

func TestWriteMakesTheFileWhereThereIsNone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "held", "numen.json")

	if err := settings.Write(path, []byte("{}\n"), nil, newDocument()); err != nil {
		t.Fatalf("writing: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the file is not there: %v", err)
	}
}

// newString is the file a caller presents as the one it last read.
func newString(output string) *string { return &output }

func TestWriteOverAFileThatMovedPastWhatWasReadIsRefused(t *testing.T) {
	read := "{\n  \"agent\": { \"use\": \"claude\" }\n}\n"
	target := "{\n  \"agent\": { \"use\": \"codex\" }\n}\n"
	path := writeSettingsFile(t, target)

	err := settings.Write(
		path, []byte(`{"agent": {"use": "gemini"}}`), newString(read), newDocument())
	if !errors.Is(err, port.ErrStale) {
		t.Fatalf("failed with %v, wanted the file to have moved past what was read", err)
	}

	raw, _ := os.ReadFile(path)
	if string(raw) != target {
		t.Errorf("the file holds %q, wanted it left as it was", raw)
	}
}

func TestWriteOverTheFileThatWasReadLands(t *testing.T) {
	target := "{\n  \"agent\": { \"use\": \"claude\" }\n}\n"
	path := writeSettingsFile(t, target)
	content := "{\n  \"agent\": { \"use\": \"gemini\" }\n}\n"

	if err := settings.Write(path, []byte(content), newString(target), newDocument()); err != nil {
		t.Fatalf("writing: %v", err)
	}

	raw, _ := os.ReadFile(path)
	if string(raw) != content {
		t.Errorf("the file holds %q, wanted %q", raw, content)
	}
}

// A file that is not there reads as an empty object, and that is what a caller
// who read it presents.
func TestWriteOverAFileThatIsNotThereLands(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numen.json")
	content := "{\n  \"agent\": { \"use\": \"gemini\" }\n}\n"

	if err := settings.Write(
		path, []byte(content), newString("{}\n"), newDocument()); err != nil {
		t.Fatalf("writing: %v", err)
	}

	raw, _ := os.ReadFile(path)
	if string(raw) != content {
		t.Errorf("the file holds %q, wanted %q", raw, content)
	}
}
