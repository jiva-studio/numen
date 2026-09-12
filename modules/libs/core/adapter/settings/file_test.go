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

// beside makes a settings file in a folder of this test's own.
func beside(t *testing.T, written string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "numen.json")
	if written != "" {
		if err := os.WriteFile(path, []byte(written), 0o600); err != nil {
			t.Fatalf("writing the file: %v", err)
		}
	}
	return path
}

func TestReadHandsBackEveryByteAsItStands(t *testing.T) {
	written := "{\n  \"agent\": { \"use\": \"claude\" }\n}\n"
	raw, err := settings.Read(beside(t, written))
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if string(raw) != written {
		t.Errorf("read back %q, wanted %q", raw, written)
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
	path := beside(t, "{}\n")
	// Two spaces of indent, a section in an order of somebody's own, and a
	// number written to two places.
	written := "{\n  \"indexing\": {\n    \"proofreading\": { \"max_edit_distance\": 0.30 }\n  }\n}\n"

	if err := settings.Write(path, []byte(written), nil); err != nil {
		t.Fatalf("writing: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading it back: %v", err)
	}
	if string(raw) != written {
		t.Errorf("the file holds %q, wanted %q", raw, written)
	}
}

func TestWriteRefusesWhatIsNotJSONAndLeavesTheFileAsItWas(t *testing.T) {
	held := "{\n  \"agent\": { \"use\": \"claude\" }\n}\n"
	path := beside(t, held)

	err := settings.Write(path, []byte("{ \"agent\": { \"use\": \"claude\" } // a comment\n}"), nil)
	if !errors.Is(err, port.ErrNotASetting) {
		t.Fatalf("refused with %v, wanted a refusal", err)
	}
	if !strings.Contains(err.Error(), "byte") {
		t.Errorf("said %q, wanted it to name where the trouble is", err)
	}

	raw, _ := os.ReadFile(path)
	if string(raw) != held {
		t.Errorf("the file holds %q, wanted it left as it was", raw)
	}
}

func TestWriteRefusesAValueOfTheWrongKind(t *testing.T) {
	path := beside(t, "{}\n")

	err := settings.Write(path, []byte(`{"appearance": {"interface_scale": "large"}}`), nil)
	if !errors.Is(err, port.ErrNotASetting) {
		t.Fatalf("refused with %v, wanted a refusal", err)
	}
	if !strings.Contains(err.Error(), "byte") {
		t.Errorf("said %q, wanted it to name where the trouble is", err)
	}
}

// A settings file is one object. Anything else read into the settings leaves
// them at their defaults, and defaults that hold say nothing about the bytes.
func TestWriteRefusesWhatIsNotAnObject(t *testing.T) {
	for _, written := range []string{"null", "42", `"hello"`, "[1,2]", "true"} {
		held := "{\n  \"agent\": { \"use\": \"claude\" }\n}\n"
		path := beside(t, held)

		err := settings.Write(path, []byte(written), nil)
		if !errors.Is(err, port.ErrNotASetting) {
			t.Errorf("%s: refused with %v, wanted a refusal", written, err)
		}

		raw, _ := os.ReadFile(path)
		if string(raw) != held {
			t.Errorf("%s: the file holds %q, wanted it left as it was", written, raw)
		}
	}
}

// A save refuses a name a section holds twice, so the editor refuses to write
// one: the settings are read from the last of the two and a save patches the
// first.
func TestWriteRefusesANameASectionHoldsTwice(t *testing.T) {
	for what, written := range map[string]string{
		"a section":        `{"appearance":{"mode":"dark"},"appearance":{"mode":"light"}}`,
		"a field":          `{"appearance":{"mode":"dark","mode":"light"}}`,
		"a field far down": `{"indexing":{"embedding":{"indexing":{"use":"a","use":"b"}}}}`,
	} {
		held := "{\n  \"agent\": { \"use\": \"claude\" }\n}\n"
		path := beside(t, held)

		err := settings.Write(path, []byte(written), nil)
		if !errors.Is(err, port.ErrNotASetting) {
			t.Errorf("%s: refused with %v, wanted a refusal", what, err)
		} else if !strings.Contains(err.Error(), "one of a name") {
			t.Errorf("%s: said %q, wanted it to name the trouble", what, err)
		}

		raw, _ := os.ReadFile(path)
		if string(raw) != held {
			t.Errorf("%s: the file holds %q, wanted it left as it was", what, raw)
		}
	}
}

func TestWhatIsSaidDoesNotRepeatWhatStandsInTheFile(t *testing.T) {
	path := beside(t, "{}\n")
	// A key is the one thing in a settings file worth keeping to itself.
	secret := "sk-not-a-real-key-0000"

	err := settings.Write(path, []byte(`{"appearance": {"interface_scale": "`+secret+`"}}`), nil)
	if err == nil {
		t.Fatal("wanted a refusal")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("said %q, which repeats what stands in the file", err)
	}
}

func TestWriteMakesTheFileWhereThereIsNone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "held", "numen.json")

	if err := settings.Write(path, []byte("{}\n"), nil); err != nil {
		t.Fatalf("writing: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the file is not there: %v", err)
	}
}

// newLastRead is the file a caller presents as the one it last read.
func newLastRead(written string) *string { return &written }

func TestWriteOverAFileThatMovedPastWhatWasReadIsRefused(t *testing.T) {
	read := "{\n  \"agent\": { \"use\": \"claude\" }\n}\n"
	held := "{\n  \"agent\": { \"use\": \"codex\" }\n}\n"
	path := beside(t, held)

	err := settings.Write(path, []byte(`{"agent": {"use": "gemini"}}`), newLastRead(read))
	if !errors.Is(err, port.ErrStale) {
		t.Fatalf("refused with %v, wanted the file to have moved past what was read", err)
	}

	raw, _ := os.ReadFile(path)
	if string(raw) != held {
		t.Errorf("the file holds %q, wanted it left as it was", raw)
	}
}

func TestWriteOverTheFileThatWasReadLands(t *testing.T) {
	held := "{\n  \"agent\": { \"use\": \"claude\" }\n}\n"
	path := beside(t, held)
	written := "{\n  \"agent\": { \"use\": \"gemini\" }\n}\n"

	if err := settings.Write(path, []byte(written), newLastRead(held)); err != nil {
		t.Fatalf("writing: %v", err)
	}

	raw, _ := os.ReadFile(path)
	if string(raw) != written {
		t.Errorf("the file holds %q, wanted %q", raw, written)
	}
}

// A file that is not there reads as an empty object, and that is what a caller
// who read it presents.
func TestWriteOverAFileThatIsNotThereLands(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numen.json")
	written := "{\n  \"agent\": { \"use\": \"gemini\" }\n}\n"

	if err := settings.Write(path, []byte(written), newLastRead("{}\n")); err != nil {
		t.Fatalf("writing: %v", err)
	}

	raw, _ := os.ReadFile(path)
	if string(raw) != written {
		t.Errorf("the file holds %q, wanted %q", raw, written)
	}
}
