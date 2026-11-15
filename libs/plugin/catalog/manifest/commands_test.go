package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const commandManifestHead = `{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"permissions":["vault:manage"],"contributes":{"commands":`

func TestAManifestDeclaresCommandsWithTheirConditionAndArguments(t *testing.T) {
	text := commandManifestHead + `[{
		"id": "pages.convert", "title": "Convert", "when": "target.contentType in ['application/pdf', 'image/vnd.djvu']",
		"permission": "vault:manage",
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
		command.When != "target.contentType in ['application/pdf', 'image/vnd.djvu']" ||
		command.Permission != manifest.PermissionVaultManage {
		t.Fatalf("the command came out as %+v", command)
	}
	pages, _ := command.GetArgument("pages")
	again, _ := command.GetArgument("again")
	if !pages.IsRequired || pages.Type != manifest.CommandArgumentNumber || again.IsRequired {
		t.Errorf("the arguments came out as %+v", command.Arguments)
	}

	withVersionArg := commandManifestHead + `[{"id":"pages.convert","title":"Convert","when":"target.isNote","arguments":[{"name":"version-id","title":"V","type":"string"}]}]}}`
	if _, err := manifestjson.Read(strings.NewReader(withVersionArg)); err != nil {
		t.Errorf("expected version-id argument without version condition to be allowed, got %v", err)
	}
}

func hasWrappedCause(err, sentinel error) bool {
	if err == nil || !errors.Is(err, sentinel) {
		return false
	}
	type multiUnwrap interface{ Unwrap() []error }
	var check func(error) bool
	check = func(current error) bool {
		if current == nil {
			return false
		}
		if multi, ok := current.(multiUnwrap); ok {
			children := multi.Unwrap()
			hasSentinel := false
			hasOther := false
			for _, child := range children {
				if errors.Is(child, sentinel) {
					hasSentinel = true
				} else if child != nil {
					hasOther = true
				}
				if check(child) {
					return true
				}
			}
			if hasSentinel && hasOther {
				return true
			}
		}
		return check(errors.Unwrap(current))
	}
	return check(err)
}

func TestACommandThatBreaksARuleIsAnInvalidCommand(t *testing.T) {
	cases := map[string]struct {
		commands    string
		want        []error
		wantSubstr  string
		forbidSub   string
		wrappedSntl error
	}{
		"an id in capitals":    {commands: `[{"id":"page.Convert","title":"C"}]`, want: []error{manifest.ErrInvalidCommand}},
		"an id with a space":   {commands: `[{"id":"page.convert pages","title":"C"}]`, want: []error{manifest.ErrInvalidCommand}},
		"an empty id":          {commands: `[{"id":"","title":"C"}]`, want: []error{manifest.ErrInvalidCommand}},
		"an id declared twice": {commands: `[{"id":"page.convert","title":"C"},{"id":"page.convert","title":"D"}]`, want: []error{manifest.ErrInvalidCommand}},
		"no title":             {commands: `[{"id":"page.convert","title":" "}]`, want: []error{manifest.ErrInvalidCommand}},
		"a condition over an undeclared variable": {
			commands: `[{"id":"page.convert","title":"C","when":"target.isRemote"}]`, want: []error{manifest.ErrInvalidCommand}, wrappedSntl: manifest.ErrInvalidCommand,
		},
		"a malformed condition": {
			commands: `[{"id":"page.convert","title":"C","when":"target.isNote &&"}]`, want: []error{manifest.ErrInvalidCommand}, wrappedSntl: manifest.ErrInvalidCommand,
		},
		"an unknown permission": {
			commands:  `[{"id":"page.convert","title":"C","permission":"vault:admin"}]`,
			want:      []error{manifest.ErrInvalidCommand, manifest.ErrInvalidPermission},
			forbidSub: "is missing from permissions",
		},
		"a permission the manifest does not declare": {
			commands:   `[{"id":"page.convert","title":"C","permission":"vault:read"}]`,
			want:       []error{manifest.ErrInvalidCommand, manifest.ErrInvalidPermission},
			wantSubstr: "is missing from permissions",
		},
		"an argument of no known type": {
			commands: `[{"id":"page.convert","title":"C","arguments":[{"name":"pages","title":"P","type":"integer"}]}]`,
			want:     []error{manifest.ErrInvalidCommandArgument},
		},
		"an argument taken twice": {
			commands: `[{"id":"page.convert","title":"C","arguments":[{"name":"p","title":"P","type":"string"},{"name":"p","title":"Q","type":"string"}]}]`,
			want:     []error{manifest.ErrInvalidCommandArgument},
		},
		"an argument named as no flag is": {
			commands: `[{"id":"page.convert","title":"C","arguments":[{"name":"--pages","title":"P","type":"string"}]}]`,
			want:     []error{manifest.ErrInvalidCommandArgument},
		},
		"an argument with no title": {
			commands: `[{"id":"page.convert","title":"C","arguments":[{"name":"pages","type":"string"}]}]`,
			want:     []error{manifest.ErrInvalidCommandArgument},
		},
		"the argument the host gives a command with items": {
			commands: `[{"id":"page.convert","title":"C","when":"version.isLive","arguments":[{"name":"version-id","title":"V","type":"string"}]}]`,
			want:     []error{manifest.ErrInvalidCommandArgument},
		},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := manifestjson.Read(strings.NewReader(commandManifestHead + one.commands + `}}`))

			for _, want := range one.want {
				if !errors.Is(err, want) {
					t.Errorf("error = %v, want %v", err, want)
				}
			}
			if one.wantSubstr != "" && !strings.Contains(err.Error(), one.wantSubstr) {
				t.Errorf("error = %v, want substring %q", err, one.wantSubstr)
			}
			if one.forbidSub != "" && strings.Contains(err.Error(), one.forbidSub) {
				t.Errorf("error = %v, did not want substring %q", err, one.forbidSub)
			}
			if one.wrappedSntl != nil && !hasWrappedCause(err, one.wrappedSntl) {
				t.Errorf("error = %v, expected wrapped cause alongside %v", err, one.wrappedSntl)
			}
		})
	}
}
