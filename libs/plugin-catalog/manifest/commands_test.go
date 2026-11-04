package manifest_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

const commandManifestHead = `{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"contributes":{"commands":`

func TestAManifestDeclaresCommandsWithTheirFilesAndArguments(t *testing.T) {
	text := commandManifestHead + `[{
		"id": "pages.convert", "title": "Convert", "contentTypes": ["application/pdf", "image/vnd.djvu"],
		"arguments": [
			{"name": "pages", "title": "Pages", "type": "number", "required": true},
			{"name": "again", "title": "Again", "type": "boolean"},
			{"name": "label", "title": "Label", "type": "string"}
		]
	}]}}`

	parsed, err := manifestjson.Read(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}

	if len(parsed.Contributes.Commands) != 1 {
		t.Fatalf("the commands came out as %+v", parsed.Contributes.Commands)
	}
	command := parsed.Contributes.Commands[0]
	if command.ID != "pages.convert" || command.Title != "Convert" || !command.HasFiles() || len(command.Arguments) != 3 {
		t.Fatalf("the command came out as %+v", command)
	}
	pages, _ := command.GetArgument("pages")
	again, _ := command.GetArgument("again")
	if !pages.IsRequired || pages.Type != manifest.CommandArgumentNumber || again.IsRequired {
		t.Errorf("the arguments came out as %+v", command.Arguments)
	}
	if !command.CanRunOn("application/pdf", "") || !command.CanRunOn("image/vnd.djvu", "") || command.CanRunOn("application/epub+zip", "") || command.CanRunOn(domain.UnknownContentType, "") {
		t.Error("the command runs over the wrong files")
	}
	if (manifest.Command{}).HasFiles() {
		t.Error("an empty command should not have files")
	}
	noteCmd := manifest.Command{NoteTypes: []string{"link"}}
	if !noteCmd.HasFiles() || !noteCmd.CanRunOn("", "link") || noteCmd.CanRunOn("", "other") {
		t.Error("note command runs over the wrong note types")
	}
}

func TestACommandOverAnyFileRunsOverEveryFile(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(
		commandManifestHead + `[{"id":"proofreading.create","title":"Proofread","contentTypes":["*/*"]}]}}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	command := m.Contributes.Commands[0]
	if !command.HasFiles() || !command.CanRunOn("audio/vnd.wave", "") || !command.CanRunOn(domain.UnknownContentType, "") {
		t.Error("the command runs over the wrong files")
	}
}

func TestACommandThatBreaksARuleIsAnInvalidCommand(t *testing.T) {
	cases := map[string]struct {
		commands string
		want     error
	}{
		"an id in capitals":       {`[{"id":"page.Convert","title":"C"}]`, manifest.ErrInvalidCommand},
		"an id with a space":      {`[{"id":"page.convert pages","title":"C"}]`, manifest.ErrInvalidCommand},
		"an empty id":             {`[{"id":"","title":"C"}]`, manifest.ErrInvalidCommand},
		"an id declared twice":    {`[{"id":"page.convert","title":"C"},{"id":"page.convert","title":"D"}]`, manifest.ErrInvalidCommand},
		"no title":                {`[{"id":"page.convert","title":" "}]`, manifest.ErrInvalidCommand},
		"an extension for a type": {`[{"id":"page.convert","title":"C","contentTypes":[".pdf"]}]`, manifest.ErrInvalidCommand},
		"a mask of a subtype":     {`[{"id":"page.convert","title":"C","contentTypes":["*/pdf"]}]`, manifest.ErrInvalidCommand},
		"a type in capitals": {
			`[{"id":"page.convert","title":"C","contentTypes":["Application/PDF"]}]`, manifest.ErrInvalidCommand,
		},
		"an argument of no known type": {
			`[{"id":"page.convert","title":"C","arguments":[{"name":"pages","title":"P","type":"integer"}]}]`,
			manifest.ErrInvalidCommandArgument,
		},
		"an argument taken twice": {
			`[{"id":"page.convert","title":"C","arguments":[{"name":"p","title":"P","type":"string"},{"name":"p","title":"Q","type":"string"}]}]`,
			manifest.ErrInvalidCommandArgument,
		},
		"an argument named as no flag is": {
			`[{"id":"page.convert","title":"C","arguments":[{"name":"--pages","title":"P","type":"string"}]}]`,
			manifest.ErrInvalidCommandArgument,
		},
		"an argument with no title": {
			`[{"id":"page.convert","title":"C","arguments":[{"name":"pages","type":"string"}]}]`,
			manifest.ErrInvalidCommandArgument,
		},
		"a required argument hidden in the palette": {
			`[{"id":"page.convert","title":"C","arguments":[{"name":"wait","title":"W","type":"boolean","required":true,"hiddenInPalette":true}]}]`,
			manifest.ErrInvalidCommandArgument,
		},
		"an undeclared note type": {
			`[{"id":"page.convert","title":"C","noteTypes":["undeclared"]}]`,
			manifest.ErrInvalidCommand,
		},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := manifestjson.Read(strings.NewReader(commandManifestHead + one.commands + `}}`))

			if !errors.Is(err, one.want) {
				t.Errorf("error = %v, want %v", err, one.want)
			}
		})
	}
}

func TestReadManifestReadsAnArgumentHiddenInThePalette(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(commandManifestHead +
		`[{"id":"link.import","title":"I","arguments":[{"name":"wait","title":"W","type":"boolean","hiddenInPalette":true}]}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if wait, _ := m.Contributes.Commands[0].GetArgument("wait"); !wait.IsHiddenInPalette {
		t.Errorf("the argument came out as %+v", wait)
	}
}

func TestManifestDeclaresCommandWithContextCondition(t *testing.T) {
	parsed, err := manifestjson.Read(strings.NewReader(commandManifestHead + `[
		{"id": "proofreading.create", "title": "Proofread", "when": [
			{"key": "version.live", "is": ["true"]}, {"key": "version.kind", "is": ["transcript", "recognition"]}]},
		{"id": "proofreading.delete", "title": "Delete proofreading", "when": [{"key": "version.plugin", "is": ["a.b"]}]}
	]}}`))
	if err != nil {
		t.Fatal(err)
	}

	proofread, deletion := parsed.Contributes.Commands[0], parsed.Contributes.Commands[1]
	wantProofread := []manifest.ContextCondition{
		{Key: "version.live", Is: []string{"true"}}, {Key: "version.kind", Is: []string{"transcript", "recognition"}},
	}
	if !slices.EqualFunc(proofread.When, wantProofread, func(a, b manifest.ContextCondition) bool { return a.Key == b.Key && slices.Equal(a.Is, b.Is) }) {
		t.Errorf("the conditions on the live version came out as %+v", proofread.When)
	}
	if len(deletion.When) != 1 || deletion.When[0].Key != "version.plugin" {
		t.Errorf("the condition on the plugin came out as %+v", deletion.When)
	}
	if !proofread.HasFiles() || !proofread.CanRunOn(domain.UnknownContentType, "") || !deletion.HasFiles() || !deletion.CanRunOn(domain.UnknownContentType, "") {
		t.Error("a command with a condition does not run over any file")
	}
}

func TestACommandRunsInTheContextsThatMeetEveryCondition(t *testing.T) {
	command := manifest.Command{When: []manifest.ContextCondition{
		{Key: "version.live", Is: []string{"true"}}, {Key: "version.kind", Is: []string{"page", "recognition"}},
	}}
	cases := map[string]struct {
		context manifest.Context
		want    bool
	}{
		"both hold":           {manifest.Context{"version.live": "true", "version.kind": "page"}, true},
		"the kind fails":      {manifest.Context{"version.live": "true", "version.kind": "transcript"}, false},
		"the liveness fails":  {manifest.Context{"version.live": "false", "version.kind": "page"}, false},
		"a key is not set":    {manifest.Context{"version.live": "true"}, false},
		"no context at all":   {nil, false},
		"another key is moot": {manifest.Context{"version.live": "true", "version.kind": "page", "other": "x"}, true},
	}
	for name, one := range cases {
		if got := command.CanRunIn(one.context); got != one.want {
			t.Errorf("%s: CanRunIn = %v, want %v", name, got, one.want)
		}
	}
	anyKind := manifest.Command{When: []manifest.ContextCondition{{Key: "version.kind", Is: []string{manifest.ContextAnyValue}}}}
	if !anyKind.CanRunIn(manifest.Context{"version.kind": "whatever"}) || anyKind.CanRunIn(manifest.Context{}) {
		t.Error("the any-value condition must hold for a key that is set, and fail for one that is not")
	}
	if !(manifest.Command{}).CanRunIn(nil) {
		t.Error("a command with no condition holds in every context")
	}
}

func TestACommandWithAnInvalidConditionIsRefused(t *testing.T) {
	for name, command := range map[string]string{
		"a condition that names no key":   `{"id":"a","title":"A","when":[{"is":["x"]}]}`,
		"a condition that names no value": `{"id":"a","title":"A","when":[{"key":"k"}]}`,
		"a condition and a content type":  `{"id":"a","title":"A","contentTypes":["audio/*"],"when":[{"key":"k","is":["x"]}]}`,
		"a condition and a note type":     `{"id":"a","title":"A","noteTypes":["link"],"when":[{"key":"k","is":["x"]}]}`,
		"a value with no name":            `{"id":"a","title":"A","when":[{"key":"k","is":[" "]}]}`,
		"the argument the host gives":     `{"id":"a","title":"A","when":[{"key":"k","is":["x"]}],"arguments":[{"name":"version-id","title":"V","type":"string"}]}`,
	} {
		head := `{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"contributes":{"noteTypes":[{"type":"link"}],"commands":`
		_, err := manifestjson.Read(strings.NewReader(head + `[` + command + `]}}`))
		if !errors.Is(err, manifest.ErrInvalidCommand) && !errors.Is(err, manifest.ErrInvalidCommandArgument) {
			t.Errorf("%s: got %v", name, err)
		}
	}
}
