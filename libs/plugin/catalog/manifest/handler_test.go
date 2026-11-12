package manifest_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const handlerManifestHead = `{"id":"org.example.scans","name":"Scans","version":"1.0.0",` +
	`"backend":{"entrypoint":"bin/scans"},"contributes":`

func TestReadManifest_ReadsTheHandlersAndTriggersOfAPlugin(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(handlerManifestHead + `{
		"handlers":[{"command":"app:file.createText","title":"Read the scan","when":"target.contentType == 'image/png'"}],
		"triggers":[{"event":"file.added","command":"app:file.createText","when":"config.org.example.scans.isAutomatic == true"}]}}`))
	if err != nil {
		t.Fatal(err)
	}

	handler, ok := m.GetHandler("app:file.createText")
	want := manifest.Handler{Command: "app:file.createText", Title: "Read the scan", When: "target.contentType == 'image/png'"}
	if !ok || handler != want {
		t.Errorf("handler = %+v, %v; want %+v", handler, ok, want)
	}
	if _, ok := m.GetHandler("app:link.import"); ok {
		t.Error("a command the plugin does not handle has a handler")
	}
	wantTriggers := []manifest.Trigger{{Event: "file.added", Command: "app:file.createText", When: "config.org.example.scans.isAutomatic == true"}}
	if !reflect.DeepEqual(m.Contributes.Triggers, wantTriggers) {
		t.Errorf("triggers = %+v; want %+v", m.Contributes.Triggers, wantTriggers)
	}
}

func TestReadManifest_RejectsAnInvalidHandler(t *testing.T) {
	for name, handlers := range map[string]string{
		"a command that is not shared": `[{"command":"org.example.scans:scan.read","title":"Read"}]`,
		"a command handled twice":      `[{"command":"app:file.createText","title":"Read"},{"command":"app:file.createText","title":"Read again"}]`,
		"no title":                     `[{"command":"app:file.createText","title":" "}]`,
		"a when that does not parse":   `[{"command":"app:file.createText","title":"Read","when":"target.isScan"}]`,
	} {
		_, err := manifestjson.Read(strings.NewReader(handlerManifestHead + `{"handlers":` + handlers + `}}`))
		if !errors.Is(err, manifest.ErrInvalidHandler) {
			t.Errorf("%s: got %v, want ErrInvalidHandler", name, err)
		}
	}
}

func TestReadManifest_RejectsAHandlerOfAPluginWithNothingToRunIt(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(`{"id":"org.example.scans","name":"Scans","version":"1.0.0",
		"contributes":{"handlers":[{"command":"app:file.createText","title":"Read"}]}}`))
	if !errors.Is(err, manifest.ErrInvalidHandler) {
		t.Errorf("got %v, want ErrInvalidHandler", err)
	}
}

func TestReadManifest_RejectsAnInvalidTrigger(t *testing.T) {
	const handlers = `"handlers":[{"command":"app:file.createText","title":"Read"}]`
	for name, triggers := range map[string]string{
		"an event that is not declared": `[{"event":"file.opened","command":"app:file.createText"}]`,
		"a command it does not handle":  `[{"event":"file.added","command":"app:link.import"}]`,
		"a when that does not parse":    `[{"event":"file.added","command":"app:file.createText","when":"target.isScan"}]`,
	} {
		_, err := manifestjson.Read(strings.NewReader(handlerManifestHead + `{` + handlers + `,"triggers":` + triggers + `}}`))
		if !errors.Is(err, manifest.ErrInvalidTrigger) {
			t.Errorf("%s: got %v, want ErrInvalidTrigger", name, err)
		}
	}
}

func TestReadManifest_RejectsATriggerOfAPluginWithoutABackend(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(`{"id":"org.example.scans","name":"Scans","version":"1.0.0","frontend":{},
		"contributes":{"handlers":[{"command":"app:file.createText","title":"Read"}],
		"triggers":[{"event":"file.added","command":"app:file.createText"}]}}`))
	if !errors.Is(err, manifest.ErrNoBackend) {
		t.Errorf("got %v, want ErrNoBackend", err)
	}
}

func TestReadManifest_ReadsTheTextKindsOfAPlugin(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(handlerManifestHead +
		`{"textKinds":[{"kind":"recognition","title":"Recognized text"}]}}`))
	if err != nil {
		t.Fatal(err)
	}

	want := []manifest.TextKind{{Kind: "recognition", Title: "Recognized text"}}
	if !reflect.DeepEqual(m.Contributes.TextKinds, want) {
		t.Errorf("text kinds = %+v; want %+v", m.Contributes.TextKinds, want)
	}
	if title, ok := m.Contributes.GetTextKindTitle("recognition"); !ok || title != "Recognized text" {
		t.Errorf("title = %q %v", title, ok)
	}
}

func TestReadManifest_RejectsATextKindWithoutAKindOrATitleOrDeclaredTwice(t *testing.T) {
	for name, kinds := range map[string]string{
		"no kind":  `[{"title":"Text"}]`,
		"no title": `[{"kind":"recognition"}]`,
		"twice":    `[{"kind":"a","title":"A"},{"kind":"a","title":"B"}]`,
	} {
		_, err := manifestjson.Read(strings.NewReader(handlerManifestHead + `{"textKinds":` + kinds + `}}`))
		if !errors.Is(err, manifest.ErrInvalidContribution) {
			t.Errorf("%s: got %v, want ErrInvalidContribution", name, err)
		}
	}
}
