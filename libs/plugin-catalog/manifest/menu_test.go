package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

const menuManifestHead = `{"id":"a.b","name":"A","version":"1.0.0","frontend":{},"contributes":{"menus":`

func readMenus(menus string) (manifest.Manifest, error) {
	return manifestjson.Read(strings.NewReader(menuManifestHead + menus + `}}`))
}

func TestAManifestDeclaresMenuEntriesWithAndWithoutGroupAndOrder(t *testing.T) {
	parsed, err := readMenus(`[
		{"location": "titleBar.commands", "command": "tree.open", "group": "navigate", "order": 10},
		{"location": "searchResult", "command": "tree.open"},
		{"location": "titleBar.commands", "command": "app:note.create", "order": 0}
	]`)
	if err != nil {
		t.Fatal(err)
	}

	menus := parsed.Contributes.Menus
	if len(menus) != 3 {
		t.Fatalf("the menus came out as %+v", menus)
	}
	first, second, third := menus[0], menus[1], menus[2]
	if first.Location != "titleBar.commands" || first.Command != "tree.open" || first.Group != "navigate" || first.Order == nil || *first.Order != 10 {
		t.Errorf("the first entry came out as %+v", first)
	}
	if second.Location != "searchResult" || second.Group != "" || second.Order != nil {
		t.Errorf("the entry without group and order came out as %+v", second)
	}
	if third.Command != "app:note.create" || third.Order == nil || *third.Order != 0 {
		t.Errorf("an order of zero came out as %+v, and must differ from no order", third)
	}
}

func TestAManifestWithoutMenusDeclaresNone(t *testing.T) {
	parsed, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","frontend":{}}`))
	if err != nil {
		t.Fatal(err)
	}

	if len(parsed.Contributes.Menus) != 0 {
		t.Errorf("the menus came out as %+v", parsed.Contributes.Menus)
	}
}

func TestAMenuEntryThatBreaksARuleIsAnInvalidMenu(t *testing.T) {
	cases := map[string]string{
		"an empty location":                 `[{"location":"","command":"tree.open"}]`,
		"a blank location":                  `[{"location":"  ","command":"tree.open"}]`,
		"no location":                       `[{"command":"tree.open"}]`,
		"an empty command":                  `[{"location":"titleBar.commands","command":""}]`,
		"a blank command":                   `[{"location":"titleBar.commands","command":" "}]`,
		"no command":                        `[{"location":"titleBar.commands"}]`,
		"a command in capitals":             `[{"location":"titleBar.commands","command":"Tree.Open"}]`,
		"a command of another plugin":       `[{"location":"titleBar.commands","command":"other.plugin:tree.open"}]`,
		"the owner of the editor alone":     `[{"location":"titleBar.commands","command":"app:"}]`,
		"an editor command in capitals":     `[{"location":"titleBar.commands","command":"app:Note.Create"}]`,
		"a group with a leading space":      `[{"location":"titleBar.commands","command":"tree.open","group":" navigate"}]`,
		"a group with a trailing space":     `[{"location":"titleBar.commands","command":"tree.open","group":"navigate "}]`,
		"the same command twice in a place": `[{"location":"titleBar.commands","command":"tree.open"},{"location":"titleBar.commands","command":"tree.open","group":"other","order":3}]`,
		"the same editor command twice":     `[{"location":"titleBar.commands","command":"app:note.create"},{"location":"titleBar.commands","command":"app:note.create"}]`,
	}
	for name, menus := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := readMenus(menus)

			if !errors.Is(err, manifest.ErrInvalidMenu) {
				t.Errorf("error = %v, want %v", err, manifest.ErrInvalidMenu)
			}
		})
	}
}

func TestAMenuEntryWithAValidCommandIsAccepted(t *testing.T) {
	for _, command := range []string{"tree.open", "app:note.create"} {
		t.Run(command, func(t *testing.T) {
			if _, err := readMenus(`[{"location":"titleBar.commands","command":"` + command + `"}]`); err != nil {
				t.Errorf("error = %v", err)
			}
		})
	}
}

func TestTheSameCommandInTwoPlacesIsValid(t *testing.T) {
	parsed, err := readMenus(`[{"location":"titleBar.commands","command":"tree.open","order":1},{"location":"searchResult","command":"tree.open","order":2}]`)
	if err != nil {
		t.Fatal(err)
	}

	if len(parsed.Contributes.Menus) != 2 {
		t.Errorf("the menus came out as %+v", parsed.Contributes.Menus)
	}
}

func TestAnOrderThatIsNotANumberIsRefused(t *testing.T) {
	for name, order := range map[string]string{"a quoted number": `"10"`, "a word": `"first"`, "a boolean": `true`, "a fraction": `1.5`} {
		t.Run(name, func(t *testing.T) {
			_, err := readMenus(`[{"location":"titleBar.commands","command":"tree.open","order":` + order + `}]`)

			if err == nil {
				t.Error("the manifest was accepted")
			}
		})
	}
}

func TestGoDoesNotJudgeTheLocationOrWhetherTheCommandIsDeclared(t *testing.T) {
	parsed, err := readMenus(`[{"location":"sidebar","command":"nothing.declared"}]`)
	if err != nil {
		t.Fatal(err)
	}

	menus := parsed.Contributes.Menus
	if len(menus) != 1 || menus[0].Location != "sidebar" || menus[0].Command != "nothing.declared" {
		t.Errorf("the menus came out as %+v", menus)
	}
}
