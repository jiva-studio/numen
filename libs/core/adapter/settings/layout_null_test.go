package settings_test

import (
	"encoding/json"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
)

func TestAListIsWrittenInsideANullLayoutOrNullSlots(t *testing.T) {
	for name, body := range map[string]string{
		"a null layout": `{"v":1,"app":{"layout":null}}`,
		"null slots":    `{"v":1,"app":{"layout":{"slots":null}}}`,
	} {
		path := write(t, body)

		err := settings.NewFiles().Save(path, newDocument(),
			settings.Setting{At: slotPath("titleBar.status"), Written: []string{tasksKey, statsKey}})

		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var app map[string]map[string]map[string][]string
		if err := json.Unmarshal(readRoot(t, path)["app"], &app); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		got := app["layout"]["slots"]["titleBar.status"]
		if len(got) != 2 || got[0] != tasksKey || got[1] != statsKey {
			t.Errorf("%s: the list in the file is %v", name, got)
		}
	}
}
