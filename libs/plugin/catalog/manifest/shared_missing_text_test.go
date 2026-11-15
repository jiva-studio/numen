package manifest_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

// missingTextCommand is the shared command that creates the text of a file that has none of its own.
const missingTextCommand = "app:file.createMissingText"

func TestTheSharedCommandsIncludeCreateMissingText(t *testing.T) {
	if !manifest.IsSharedCommand(missingTextCommand) {
		t.Errorf("%s is not a shared command", missingTextCommand)
	}
	if !manifest.IsSharedCommand("app:file.createText") {
		t.Error("app:file.createText stopped being a shared command")
	}
}

func TestAPluginHandlesCreateMissingTextAndRunsItFromATrigger(t *testing.T) {
	parsed, err := manifestjson.Read(strings.NewReader(handlerManifestHead + `{
		"handlers":[{"command":"` + missingTextCommand + `","title":"Read the scan"}],
		"triggers":[{"event":"file.added","command":"` + missingTextCommand + `"}]}}`))
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := parsed.GetHandler(missingTextCommand); !ok {
		t.Errorf("the plugin has no handler of %s", missingTextCommand)
	}
	want := []manifest.Trigger{{Event: "file.added", Command: missingTextCommand}}
	if !reflect.DeepEqual(parsed.Contributes.Triggers, want) {
		t.Errorf("triggers = %+v; want %+v", parsed.Contributes.Triggers, want)
	}
}
