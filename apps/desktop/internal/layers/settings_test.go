package layers

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"
)

// paths is the table the settings page renders its rows from. It lives in its
// own file, so this test reads the same table as the window.
const paths = "../../editor/src/pages/settings/paths.json"

// Every path the settings page reads is a path through the file this build
// writes. A row reading a key nothing writes draws nothing, whatever the file
// holds, and writing that row puts a key in the file that nothing acts on.
func TestPageSettingsPaths_PersistedInFile(t *testing.T) {
	raw, err := os.ReadFile(paths)
	if err != nil {
		t.Fatal(err)
	}
	var settingsData map[string][]string
	if err := json.Unmarshal(raw, &settingsData); err != nil {
		t.Fatal(err)
	}
	if len(settingsData) == 0 {
		t.Fatal("the page draws no rows")
	}

	content, err := container.DefaultSettings().WriteJSON()
	if err != nil {
		t.Fatal(err)
	}
	var target any
	if err := json.Unmarshal([]byte(content), &target); err != nil {
		t.Fatal(err)
	}

	for row, item := range settingsData {
		value := target
		ok := true
		for _, step := range item {
			object, is := value.(map[string]any)
			if !is {
				ok = false
				break
			}
			value, ok = object[step]
			if !ok {
				break
			}
		}
		if !ok {
			t.Errorf("%s reads %v, which is no path through the file", row, item)
		}
	}
}
