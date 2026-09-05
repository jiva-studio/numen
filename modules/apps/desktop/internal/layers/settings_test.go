package layers

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
)

// paths is the table the settings page draws its rows from, which it holds in a
// file of its own so that this can be read against the same table the window
// reads.
const paths = "../../ui/src/settings/paths.json"

// Every path the settings page reads is a path through the file this build
// writes. A row reading a key nothing writes draws nothing, whatever the file
// holds, and writing that row puts a key in the file that nothing acts on.
func TestEveryPathThePageReadsStandsInTheFile(t *testing.T) {
	raw, err := os.ReadFile(paths)
	if err != nil {
		t.Fatal(err)
	}
	var drawn map[string][]string
	if err := json.Unmarshal(raw, &drawn); err != nil {
		t.Fatal(err)
	}
	if len(drawn) == 0 {
		t.Fatal("the page draws no rows")
	}

	written, err := settings.Written(settings.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	var held any
	if err := json.Unmarshal([]byte(written), &held); err != nil {
		t.Fatal(err)
	}

	for row, at := range drawn {
		value := held
		ok := true
		for _, step := range at {
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
			t.Errorf("%s reads %v, which is no path through the file", row, at)
		}
	}
}
