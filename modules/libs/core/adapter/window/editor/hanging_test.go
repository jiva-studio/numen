package editor_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// The window asks whether a node hangs the headings of its note, and turns the
// setting from the palette.

// hangs is whether a node hangs the headings of its note under it, and parts
// how many of them stand there at once, as the settings hold the two.
func hangs(t *testing.T, f *going) bool {
	t.Helper()
	held, is := getSetting(t, f, "appearance", "hang_parts_under_a_node").(bool)
	if !is {
		t.Fatal("the setting is not written as a switch")
	}
	return held
}

func parts(t *testing.T, f *going) int {
	t.Helper()
	held, is := getSetting(t, f, "appearance", "parts_under_a_node").(float64)
	if !is {
		t.Fatal("the count is not written as a number")
	}
	return int(held)
}

// turnsHanging writes the switch into the settings, with the count beside it
// where one is named.
func turnsHanging(t *testing.T, f *going, hangs bool, count *int) error {
	t.Helper()
	written := []*v1.Setting{
		{At: []string{"appearance", "hang_parts_under_a_node"}, Value: fmt.Sprint(hangs)},
	}
	if count != nil {
		written = append(written, &v1.Setting{
			At: []string{"appearance", "parts_under_a_node"}, Value: fmt.Sprint(*count),
		})
	}
	_, err := f.configuring.WriteSettings(t.Context(),
		connect.NewRequest(&v1.WriteSettingsRequest{Settings: written}))
	return err
}

// TestTurningTheHangingIsAnsweredByTheNextQuestion. The palette turns it, the
// file is written, and the window reads what was written. Nothing is launched
// again in between.
func TestTurningTheHangingIsAnsweredByTheNextQuestion(t *testing.T) {
	f := opening(t, nil, nil, true)

	if !hangs(t, f) {
		t.Fatal("an installation nobody has configured hangs nothing")
	}

	if err := turnsHanging(t, f, false, nil); err != nil {
		t.Fatal(err)
	}

	if hangs(t, f) {
		t.Error("the setting was turned and the window still hangs the parts")
	}
}

// How many parts stand under a node is turned the same way, and a request
// naming no count leaves the one the settings hold where it was.
func TestTurningTheCountOfPartsIsAnsweredByTheNextQuestion(t *testing.T) {
	f := opening(t, nil, nil, true)

	if said := parts(t, f); said != 6 {
		t.Fatalf("an installation nobody has configured stands %d", said)
	}

	three := 3
	if err := turnsHanging(t, f, true, &three); err != nil {
		t.Fatal(err)
	}

	if err := turnsHanging(t, f, false, nil); err != nil {
		t.Fatal(err)
	}

	if said := parts(t, f); said != 3 {
		t.Errorf("a node stands %d parts", said)
	}
	if hangs(t, f) {
		t.Error("the setting was turned and the window still hangs the parts")
	}
}

// A count the setting does not take is not written, and the settings are left
// as they are.
func TestACountOfPartsOutsideWhatItGoesToIsNotWritten(t *testing.T) {
	f := opening(t, nil, nil, true)

	twenty := 20
	if err := turnsHanging(t, f, true, &twenty); err == nil {
		t.Fatal("a count of twenty was written")
	}

	if said := parts(t, f); said != 6 {
		t.Errorf("a node stands %d parts", said)
	}
}

// The setting a person turns is written where they will read it, and every
// other byte of the file is left as they typed it.
func TestTurningTheHangingLeavesTheRestOfTheFileAlone(t *testing.T) {
	f := opening(t, nil, nil, true)
	path := filepath.Join(filepath.Dir(f.settings), "numen.json")
	if err := os.WriteFile(path, []byte("{\n  \"appearance\": {\"text_scale\": 1.5}\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := turnsHanging(t, f, false, nil); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"text_scale": 1.5`) {
		t.Errorf("what the person typed was rewritten:\n%s", raw)
	}
	if !strings.Contains(string(raw), `"hang_parts_under_a_node": false`) {
		t.Errorf("the setting is not in the file:\n%s", raw)
	}
}
