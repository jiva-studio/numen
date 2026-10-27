package pluginport_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/notes/domain"
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

	manifest, err := pluginport.ReadManifest(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}

	if len(manifest.Commands) != 1 {
		t.Fatalf("the commands came out as %+v", manifest.Commands)
	}
	command := manifest.Commands[0]
	if command.ID != "convert-pages" || command.Title != "Convert" || !command.IsHiddenInPalette || !command.HasFiles() || len(command.Arguments) != 3 {
		t.Fatalf("the command came out as %+v", command)
	}
	pages, _ := command.GetArgument("pages")
	again, _ := command.GetArgument("again")
	if !pages.IsRequired || pages.Type != pluginport.CommandArgumentNumber || again.IsRequired {
		t.Errorf("the arguments came out as %+v", command.Arguments)
	}
	if !command.CanRunOn("application/pdf", "") || !command.CanRunOn("image/vnd.djvu", "") || command.CanRunOn("application/epub+zip", "") || command.CanRunOn(domain.UnknownContentType, "") {
		t.Error("the command runs over the wrong files")
	}
}

func TestACommandOverAnyFileRunsOverEveryFile(t *testing.T) {
	m, err := pluginport.ReadManifest(strings.NewReader(
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
		"an id in capitals":       {`[{"id":"Convert","title":"C"}]`, pluginport.ErrInvalidCommand},
		"an id with a space":      {`[{"id":"convert pages","title":"C"}]`, pluginport.ErrInvalidCommand},
		"an empty id":             {`[{"id":"","title":"C"}]`, pluginport.ErrInvalidCommand},
		"an id declared twice":    {`[{"id":"convert","title":"C"},{"id":"convert","title":"D"}]`, pluginport.ErrInvalidCommand},
		"no title":                {`[{"id":"convert","title":" "}]`, pluginport.ErrInvalidCommand},
		"an extension for a type": {`[{"id":"convert","title":"C","contentTypes":[".pdf"]}]`, pluginport.ErrInvalidCommand},
		"a mask of a subtype":     {`[{"id":"convert","title":"C","contentTypes":["*/pdf"]}]`, pluginport.ErrInvalidCommand},
		"a type in capitals": {
			`[{"id":"convert","title":"C","contentTypes":["Application/PDF"]}]`, pluginport.ErrInvalidCommand,
		},
		"an argument of no known type": {
			`[{"id":"convert","title":"C","arguments":[{"name":"pages","title":"P","type":"integer"}]}]`,
			pluginport.ErrInvalidCommandArgument,
		},
		"an argument taken twice": {
			`[{"id":"convert","title":"C","arguments":[{"name":"p","title":"P","type":"string"},{"name":"p","title":"Q","type":"string"}]}]`,
			pluginport.ErrInvalidCommandArgument,
		},
		"an argument named as no flag is": {
			`[{"id":"convert","title":"C","arguments":[{"name":"--pages","title":"P","type":"string"}]}]`,
			pluginport.ErrInvalidCommandArgument,
		},
		"an argument with no title": {
			`[{"id":"convert","title":"C","arguments":[{"name":"pages","type":"string"}]}]`,
			pluginport.ErrInvalidCommandArgument,
		},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := pluginport.ReadManifest(strings.NewReader(commandManifestHead + one.commands + `}`))

			if !errors.Is(err, one.want) {
				t.Errorf("error = %v, want %v", err, one.want)
			}
		})
	}
}

func TestAPluginThatDeclaresCommandsHasABackendToRunThem(t *testing.T) {
	text := `{"id":"a.b","name":"A","version":"1.0.0","commands":[{"id":"convert","title":"C"}]}`

	_, err := pluginport.ReadManifest(strings.NewReader(text))

	if !errors.Is(err, pluginport.ErrInvalidCommand) || !strings.Contains(err.Error(), "no backend") {
		t.Errorf("error = %v, want a command with no backend", err)
	}
}
