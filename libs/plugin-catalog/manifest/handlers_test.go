package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

const handlerManifestHead = `{"id":"org.example.ledger","name":"Ledger","version":"1.0.0","backend":{"entrypoint":"bin/ledger"},"contributes":`

func readHandlers(contributes string) (manifest.Manifest, error) {
	return manifestjson.Read(strings.NewReader(handlerManifestHead + contributes + `}`))
}

func TestAManifestDeclaresHandlersOfSharedCommandsAndTheirTriggers(t *testing.T) {
	parsed, err := readHandlers(`{
		"handlers": [
			{"command": "app:file.createText", "title": "Read the ledger", "when": "target.contentType == 'application/pdf'"},
			{"command": "app:version.delete", "title": "Delete the ledger text", "when": "version.kind == 'ledger'"}
		],
		"triggers": [{"on": "file.added", "command": "app:file.createText", "when": "config.org.example.ledger.isOn == true"}]
	}`)
	if err != nil {
		t.Fatal(err)
	}

	handlers, triggers := parsed.Contributes.Handlers, parsed.Contributes.Triggers
	if len(handlers) != 2 || handlers[0] != (manifest.Handler{
		Command: "app:file.createText", Title: "Read the ledger", When: "target.contentType == 'application/pdf'",
	}) {
		t.Errorf("the handlers came out as %+v", handlers)
	}
	if len(triggers) != 1 || triggers[0] != (manifest.Trigger{
		On: "file.added", Command: "app:file.createText", When: "config.org.example.ledger.isOn == true",
	}) {
		t.Errorf("the triggers came out as %+v", triggers)
	}
	if handler, isFound := parsed.GetHandler("app:version.delete"); !isFound || handler.Title != "Delete the ledger text" {
		t.Errorf("GetHandler = %+v, %v", handler, isFound)
	}
}

func TestAHandlerOrATriggerThatBreaksARuleIsAnInvalidContribution(t *testing.T) {
	cases := map[string]string{
		"a command that is not shared":  `{"handlers":[{"command":"app:file.burn","title":"Burn"}]}`,
		"a command of a plugin":         `{"handlers":[{"command":"org.example.ledger:ledger.open","title":"Open"}]}`,
		"a handler without a title":     `{"handlers":[{"command":"app:file.createText","title":" "}]}`,
		"a command handled twice":       `{"handlers":[{"command":"app:file.createText","title":"A"},{"command":"app:file.createText","title":"B"}]}`,
		"a handler of a malformed when": `{"handlers":[{"command":"app:file.createText","title":"A","when":"target.isNote &&"}]}`,
		"a trigger on no known event":   `{"handlers":[{"command":"app:file.createText","title":"A"}],"triggers":[{"on":"file.removed","command":"app:file.createText"}]}`,
		"a trigger of no own handler":   `{"triggers":[{"on":"file.added","command":"app:file.createText"}]}`,
		"a trigger of a malformed when": `{"handlers":[{"command":"app:file.createText","title":"A"}],"triggers":[{"on":"file.added","command":"app:file.createText","when":"("}]}`,
	}
	for name, contributes := range cases {
		_, err := readHandlers(contributes)

		if !errors.Is(err, manifest.ErrInvalidHandler) {
			t.Errorf("%s: error = %v, want an invalid handler", name, err)
		}
	}
}
