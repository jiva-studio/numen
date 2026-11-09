package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const commandManifestHead = `{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"contributes":{"commands":`

func TestAManifestDeclaresCommandsWithTheirConditionAndArguments(t *testing.T) {
	text := commandManifestHead + `[{
		"id": "pages.convert", "title": "Convert", "when": "target.contentType in ['application/pdf', 'image/vnd.djvu']",
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
	if command.ID != "pages.convert" || command.Title != "Convert" || len(command.Arguments) != 3 ||
		command.When != "target.contentType in ['application/pdf', 'image/vnd.djvu']" {
		t.Fatalf("the command came out as %+v", command)
	}
	pages, _ := command.GetArgument("pages")
	again, _ := command.GetArgument("again")
	if !pages.IsRequired || pages.Type != manifest.CommandArgumentNumber || again.IsRequired {
		t.Errorf("the arguments came out as %+v", command.Arguments)
	}
}

func TestACommandThatBreaksARuleIsAnInvalidCommand(t *testing.T) {
	cases := map[string]struct {
		commands string
		want     error
	}{
		"an id in capitals":    {`[{"id":"page.Convert","title":"C"}]`, manifest.ErrInvalidCommand},
		"an id with a space":   {`[{"id":"page.convert pages","title":"C"}]`, manifest.ErrInvalidCommand},
		"an empty id":          {`[{"id":"","title":"C"}]`, manifest.ErrInvalidCommand},
		"an id declared twice": {`[{"id":"page.convert","title":"C"},{"id":"page.convert","title":"D"}]`, manifest.ErrInvalidCommand},
		"no title":             {`[{"id":"page.convert","title":" "}]`, manifest.ErrInvalidCommand},
		"a condition over an undeclared variable": {
			`[{"id":"page.convert","title":"C","when":"target.isRemote"}]`, manifest.ErrInvalidCommand,
		},
		"a malformed condition": {`[{"id":"page.convert","title":"C","when":"target.isNote &&"}]`, manifest.ErrInvalidCommand},
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
		"the argument the host gives a command with items": {
			`[{"id":"page.convert","title":"C","when":"version.isLive","arguments":[{"name":"version-id","title":"V","type":"string"}]}]`,
			manifest.ErrInvalidCommandArgument,
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
