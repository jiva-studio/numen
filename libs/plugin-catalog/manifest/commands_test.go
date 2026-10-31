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

const commandManifestHead = `{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"commands":`

func TestAManifestDeclaresCommandsWithTheirFilesAndArguments(t *testing.T) {
	text := commandManifestHead + `[{
		"id": "convert-pages", "title": "Convert", "contentTypes": ["application/pdf", "image/vnd.djvu"], "hiddenInPalette": true,
		"arguments": [
			{"name": "pages", "title": "Pages", "type": "number", "required": true},
			{"name": "again", "title": "Again", "type": "boolean"},
			{"name": "label", "title": "Label", "type": "string"}
		]
	}]}`

	parsed, err := manifestjson.Read(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}

	if len(parsed.Commands) != 1 {
		t.Fatalf("the commands came out as %+v", parsed.Commands)
	}
	command := parsed.Commands[0]
	if command.ID != "convert-pages" || command.Title != "Convert" || !command.IsHiddenInPalette || !command.HasFiles() || len(command.Arguments) != 3 {
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
}

func TestACommandOverAnyFileRunsOverEveryFile(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(
		commandManifestHead + `[{"id":"proofread","title":"Proofread","contentTypes":["*/*"]}]}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	command := m.Commands[0]
	if !command.HasFiles() || !command.CanRunOn("audio/vnd.wave", "") || !command.CanRunOn(domain.UnknownContentType, "") {
		t.Error("the command runs over the wrong files")
	}
}

func TestACommandThatBreaksARuleIsAnInvalidCommand(t *testing.T) {
	cases := map[string]struct {
		commands string
		want     error
	}{
		"an id in capitals":       {`[{"id":"Convert","title":"C"}]`, manifest.ErrInvalidCommand},
		"an id with a space":      {`[{"id":"convert pages","title":"C"}]`, manifest.ErrInvalidCommand},
		"an empty id":             {`[{"id":"","title":"C"}]`, manifest.ErrInvalidCommand},
		"an id declared twice":    {`[{"id":"convert","title":"C"},{"id":"convert","title":"D"}]`, manifest.ErrInvalidCommand},
		"no title":                {`[{"id":"convert","title":" "}]`, manifest.ErrInvalidCommand},
		"an extension for a type": {`[{"id":"convert","title":"C","contentTypes":[".pdf"]}]`, manifest.ErrInvalidCommand},
		"a mask of a subtype":     {`[{"id":"convert","title":"C","contentTypes":["*/pdf"]}]`, manifest.ErrInvalidCommand},
		"a type in capitals": {
			`[{"id":"convert","title":"C","contentTypes":["Application/PDF"]}]`, manifest.ErrInvalidCommand,
		},
		"an argument of no known type": {
			`[{"id":"convert","title":"C","arguments":[{"name":"pages","title":"P","type":"integer"}]}]`,
			manifest.ErrInvalidCommandArgument,
		},
		"an argument taken twice": {
			`[{"id":"convert","title":"C","arguments":[{"name":"p","title":"P","type":"string"},{"name":"p","title":"Q","type":"string"}]}]`,
			manifest.ErrInvalidCommandArgument,
		},
		"an argument named as no flag is": {
			`[{"id":"convert","title":"C","arguments":[{"name":"--pages","title":"P","type":"string"}]}]`,
			manifest.ErrInvalidCommandArgument,
		},
		"an argument with no title": {
			`[{"id":"convert","title":"C","arguments":[{"name":"pages","type":"string"}]}]`,
			manifest.ErrInvalidCommandArgument,
		},
		"a required argument hidden in the palette": {
			`[{"id":"convert","title":"C","arguments":[{"name":"wait","title":"W","type":"boolean","required":true,"hiddenInPalette":true}]}]`,
			manifest.ErrInvalidCommandArgument,
		},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := manifestjson.Read(strings.NewReader(commandManifestHead + one.commands + `}`))

			if !errors.Is(err, one.want) {
				t.Errorf("error = %v, want %v", err, one.want)
			}
		})
	}
}

func TestAPluginThatDeclaresCommandsHasABackendToRunThem(t *testing.T) {
	text := `{"id":"a.b","name":"A","version":"1.0.0","commands":[{"id":"convert","title":"C"}]}`

	_, err := manifestjson.Read(strings.NewReader(text))

	if !errors.Is(err, manifest.ErrInvalidCommand) || !strings.Contains(err.Error(), "no backend") {
		t.Errorf("error = %v, want a command with no backend", err)
	}
}

func TestReadManifestReadsAnArgumentHiddenInThePalette(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(commandManifestHead +
		`[{"id":"import","title":"I","arguments":[{"name":"wait","title":"W","type":"boolean","hiddenInPalette":true}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if wait, _ := m.Commands[0].GetArgument("wait"); !wait.IsHiddenInPalette {
		t.Errorf("the argument came out as %+v", wait)
	}
}

func TestManifestDeclaresCommandWithVersionCondition(t *testing.T) {
	parsed, err := manifestjson.Read(strings.NewReader(commandManifestHead + `[
		{"id": "proofread", "title": "Proofread", "when": {"liveTextKinds": ["transcript", "recognition"]}},
		{"id": "delete-proofreading", "title": "Delete proofreading", "when": {"hasOwnText": true}}
	]}`))
	if err != nil {
		t.Fatal(err)
	}

	proofread, deletion := parsed.Commands[0], parsed.Commands[1]
	if proofread.When == nil || !slices.Equal(proofread.When.LiveTextKinds, []domain.TextKind{"transcript", "recognition"}) || proofread.When.HasOwnText {
		t.Errorf("the condition on the live kind came out as %+v", proofread.When)
	}
	if deletion.When == nil || !deletion.When.HasOwnText || len(deletion.When.LiveTextKinds) != 0 {
		t.Errorf("the condition on the own text came out as %+v", deletion.When)
	}
	if !proofread.HasFiles() || !proofread.CanRunOn(domain.UnknownContentType, "") {
		t.Error("a command with a condition does not run over any file")
	}
}

func TestACommandWithAnInvalidConditionIsRefused(t *testing.T) {
	for name, command := range map[string]string{
		"a condition that names nothing": `{"id":"a","title":"A","when":{}}`,
		"a condition and a content type": `{"id":"a","title":"A","contentTypes":["audio/*"],"when":{"hasOwnText":true}}`,
		"a kind with no name":            `{"id":"a","title":"A","when":{"liveTextKinds":[" "]}}`,
		"the argument the host gives":    `{"id":"a","title":"A","when":{"hasOwnText":true},"arguments":[{"name":"version-id","title":"V","type":"string"}]}`,
	} {
		_, err := manifestjson.Read(strings.NewReader(commandManifestHead + `[` + command + `]}`))
		if !errors.Is(err, manifest.ErrInvalidCommand) && !errors.Is(err, manifest.ErrInvalidCommandArgument) {
			t.Errorf("%s: got %v", name, err)
		}
	}
}
