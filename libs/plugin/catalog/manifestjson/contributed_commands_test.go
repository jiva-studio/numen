package manifestjson_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const contributedManifestHead = `{"id":"a.b","name":"A","version":"1.0.0"`

func TestReadManifest_ContributedCommands(t *testing.T) {
	text := contributedManifestHead + `,"frontend":{},"contributes":{"commands":[
		{"id":"tree.revealNote","title":"Show note in file tree","icon":"folder-open","when":"target.isNote"}
	]}}`

	parsed, err := manifestjson.Read(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}

	if len(parsed.Contributes.Commands) != 1 {
		t.Fatalf("the commands came out as %+v", parsed.Contributes.Commands)
	}
	command := parsed.Contributes.Commands[0]
	if command.ID != "tree.revealNote" || command.Title != "Show note in file tree" || command.Icon != "folder-open" {
		t.Errorf("the command came out as %+v", command)
	}
	if command.When != "target.isNote" {
		t.Errorf("the command lost its condition: %+v", command)
	}
}

func TestACommandAtTheTopLevelOfAManifestIsDroppedWithAWarning(t *testing.T) {
	text := contributedManifestHead + `,"backend":{"entrypoint":"bin/a"},"commands":[{"id":"deck.create","title":"Create deck"}]}`

	parsed, err := manifestjson.Read(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}

	if len(parsed.Contributes.Commands) != 0 {
		t.Errorf("the top-level commands were read: %+v", parsed.Contributes.Commands)
	}
	if !slices.Contains(parsed.Warnings, `unknown key "commands"`) {
		t.Errorf("the warnings were %v", parsed.Warnings)
	}
}

func TestACommandIdDeclaredTwiceMakesTheManifestInvalid(t *testing.T) {
	text := contributedManifestHead + `,"frontend":{},"contributes":{"commands":[
		{"id":"tree.open","title":"Open file tree"},
		{"id":"tree.open","title":"Open file tree again"}
	]}}`

	_, err := manifestjson.Read(strings.NewReader(text))

	if !errors.Is(err, manifest.ErrInvalidCommand) {
		t.Fatalf("the manifest read as %v", err)
	}
}

func TestAPluginWithoutABackendDeclaresCommands(t *testing.T) {
	text := contributedManifestHead + `,"frontend":{},"contributes":{"commands":[{"id":"tree.open","title":"Open file tree"}]}}`

	parsed, err := manifestjson.Read(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}

	if len(parsed.Contributes.Commands) != 1 {
		t.Errorf("the commands came out as %+v", parsed.Contributes.Commands)
	}
}

func TestAPluginWithNeitherAFrontendNorABackendMayDeclareCommands(t *testing.T) {
	text := contributedManifestHead + `,"contributes":{"commands":[{"id":"tree.open","title":"Open file tree"}]}}`

	parsed, err := manifestjson.Read(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}

	if len(parsed.Contributes.Commands) != 1 {
		t.Errorf("the commands came out as %+v", parsed.Contributes.Commands)
	}
}
