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

// setting from the palette.

// getHeadingLimit how many of them stand there at once, as the settings hold the two.
func isHanging(t *testing.T, f *going) bool {
	t.Helper()
	target, is := getSetting(t, f, "appearance", "nest_headings_under_notes").(bool)
	if !is {
		t.Fatal("the setting is not written as a switch")
	}
	return target
}

func getHeadingLimit(t *testing.T, f *going) int {
	t.Helper()
	target, is := getSetting(t, f, "appearance", "nested_heading_limit").(float64)
	if !is {
		t.Fatal("the count is not written as a number")
	}
	return int(target)
}

// turnsHanging writes the switch into the settings, with the count beside it
// where one is named.
func turnsHanging(t *testing.T, f *going, hangs bool, count *int) error {
	t.Helper()
	content := []*v1.Setting{
		{At: []string{"appearance", "nest_headings_under_notes"}, Value: fmt.Sprint(hangs)},
	}
	if count != nil {
		content = append(content, &v1.Setting{
			At: []string{"appearance", "nested_heading_limit"}, Value: fmt.Sprint(*count),
		})
	}
	_, err := f.configuring.WriteSettings(t.Context(),
		connect.NewRequest(&v1.WriteSettingsRequest{Settings: content}))
	return err
}

// TestTurningTheNestingIsAnsweredByTheNextQuestion. The palette turns it, the
// file is written, and the window reads what was written. Nothing is launched
// again in between.
func TestTurningTheNestingIsAnsweredByTheNextQuestion(t *testing.T) {
	f := opening(t, nil, nil, true)

	if !isHanging(t, f) {
		t.Fatal("an installation nobody has configured hangs nothing")
	}

	if err := turnsHanging(t, f, false, nil); err != nil {
		t.Fatal(err)
	}

	if isHanging(t, f) {
		t.Error("the setting was turned and the window still hangs the headings")
	}
}

// How many headings stand under a node is turned the same way, and a request
// naming no count leaves the one the settings hold where it was.
func TestTurningTheCountOfHeadingsIsAnsweredByTheNextQuestion(t *testing.T) {
	f := opening(t, nil, nil, true)

	if response := getHeadingLimit(t, f); response != 6 {
		t.Fatalf("an installation nobody has configured stands %d", response)
	}

	three := 3
	if err := turnsHanging(t, f, true, &three); err != nil {
		t.Fatal(err)
	}

	if err := turnsHanging(t, f, false, nil); err != nil {
		t.Fatal(err)
	}

	if responseVal := getHeadingLimit(t, f); responseVal != 3 {
		t.Errorf("a node stands %d headings", responseVal)
	}
	if isHanging(t, f) {
		t.Error("the setting was turned and the window still hangs the headings")
	}
}

// A count the setting does not take is not written, and the settings are left
// as they are.
func TestACountOfHeadingsOutsideWhatItGoesToIsNotSaved(t *testing.T) {
	f := opening(t, nil, nil, true)

	twenty := 20
	if err := turnsHanging(t, f, true, &twenty); err == nil {
		t.Fatal("a count of twenty was written")
	}

	if response := getHeadingLimit(t, f); response != 6 {
		t.Errorf("a node stands %d headings", response)
	}
}

// The setting a person turns is written where they will read it, and every
// other byte of the file is left as they typed it.
func TestTurningTheNestingPreservesTheRestOfTheFileAlone(t *testing.T) {
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
	if !strings.Contains(string(raw), `"nest_headings_under_notes": false`) {
		t.Errorf("the setting is not in the file:\n%s", raw)
	}
}
