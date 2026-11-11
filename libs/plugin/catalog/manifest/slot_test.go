package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const slotManifestHead = `{"id":"org.example.ledger","name":"Ledger","version":"1.0.0","frontend":{},"contributes":{
	"commands":[{"id":"ledger.open","title":"Open ledger","icon":"book"},{"id":"ledger.sum","title":"Sum"}],"slots":`

func readSlots(slots string) (manifest.Manifest, error) {
	return manifestjson.Read(strings.NewReader(slotManifestHead + slots + `}}`))
}

func TestAManifestPlacesCommandsViewsAndSubmenusInSlots(t *testing.T) {
	parsed, err := readSlots(`[
		{"slot": "titleBar.commands", "command": "ledger.open", "group": "navigate", "order": 10},
		{"slot": "titleBar.status", "view": "balance"},
		{"slot": "fileTree.context", "submenu": "ledger.new", "title": "New", "icon": "plus", "group": "0-new"},
		{"slot": "ledger.new", "command": "ledger.sum", "order": 0},
		{"slot": "otherPlugin.menu", "command": "app:note.create"}
	]`)
	if err != nil {
		t.Fatal(err)
	}

	slots := parsed.Contributes.Slots
	if len(slots) != 5 {
		t.Fatalf("the slots came out as %+v", slots)
	}
	if one := slots[0]; one.Slot != "titleBar.commands" || one.Command != "ledger.open" || one.Group != "navigate" || one.Order == nil || *one.Order != 10 {
		t.Errorf("the command entry came out as %+v", one)
	}
	if one := slots[1]; one.View != "balance" || one.Command != "" || one.Order != nil {
		t.Errorf("the view entry came out as %+v", one)
	}
	if one := slots[2]; one.Submenu != "ledger.new" || one.Title != "New" || one.Icon != "plus" || one.Group != "0-new" {
		t.Errorf("the submenu entry came out as %+v", one)
	}
	if one := slots[3]; one.Slot != "ledger.new" || one.Order == nil || *one.Order != 0 {
		t.Errorf("an order of zero came out as %+v, and must differ from no order", one)
	}
}

func TestASlotEntryThatBreaksARuleIsAnInvalidSlotEntry(t *testing.T) {
	cases := map[string]string{
		"no slot":                             `[{"command":"ledger.open"}]`,
		"a blank slot":                        `[{"slot":" ","command":"ledger.open"}]`,
		"nothing placed":                      `[{"slot":"titleBar.commands"}]`,
		"a command it does not declare":       `[{"slot":"titleBar.commands","command":"ledger.close"}]`,
		"a command that is no id":             `[{"slot":"titleBar.commands","command":"Ledger Open"}]`,
		"a view id in capitals":               `[{"slot":"titleBar.status","view":"Balance"}]`,
		"a submenu without a title":           `[{"slot":"fileTree.context","submenu":"ledger.new"}]`,
		"a submenu id that is no id":          `[{"slot":"fileTree.context","submenu":"New","title":"New"}]`,
		"a submenu named as a slot":           `[{"slot":"fileTree.context","submenu":"titleBar.status","title":"New"}]`,
		"a submenu declared twice":            `[{"slot":"fileTree.context","submenu":"ledger.new","title":"New"},{"slot":"tabStrip.context","submenu":"ledger.new","title":"New"}]`,
		"a submenu icon that is no icon name": `[{"slot":"fileTree.context","submenu":"ledger.new","title":"New","icon":"Plus"}]`,
		"the same command twice in a slot":    `[{"slot":"titleBar.commands","command":"ledger.open"},{"slot":"titleBar.commands","command":"ledger.open","order":2}]`,
		"the same view twice in a slot":       `[{"slot":"titleBar.status","view":"balance"},{"slot":"titleBar.status","view":"balance"}]`,
		"a group with surrounding space":      `[{"slot":"titleBar.commands","command":"ledger.open","group":" navigate"}]`,
		"two views in the title bar":          `[{"slot":"titleBar","view":"bar"},{"slot":"titleBar","view":"strip"}]`,
		"an order written as text":            `[{"slot":"titleBar.commands","command":"ledger.open","order":"1"}]`,
		"a view in a plugin without frontend": ``,
	}
	for name, slots := range cases {
		var err error
		if slots == "" {
			_, err = manifestjson.Read(strings.NewReader(
				`{"id":"a.b","name":"A","version":"1.0.0","contributes":{"slots":[{"slot":"titleBar","view":"bar"}]}}`))
		} else {
			_, err = readSlots(slots)
		}

		if !errors.Is(err, manifest.ErrInvalidSlotEntry) && !errors.Is(err, manifest.ErrInvalidContribution) {
			t.Errorf("%s: error = %v, want an invalid slot entry", name, err)
		}
	}
}

func TestASlotEntryMayNameTheFullIdOfACommandOfAnotherOwner(t *testing.T) {
	for _, command := range []string{"app:settings.open", "md.numen.views.file.tree:tree.open"} {
		if _, err := readSlots(`[{"slot":"titleBar.commands","command":"` + command + `"}]`); err != nil {
			t.Errorf("%s: %v", command, err)
		}
	}
}

func TestTheSameCommandMayGoIntoTwoSlots(t *testing.T) {
	_, err := readSlots(`[{"slot":"titleBar.commands","command":"ledger.open"},{"slot":"fileTree.context","command":"ledger.open"}]`)

	if err != nil {
		t.Errorf("error = %v, want none", err)
	}
}

const valuesManifestHead = `{"id":"org.example.ledger","name":"Ledger","version":"1.0.0","frontend":{},"contributes":{
	"commands":[
		{"id":"ledger.open","title":"Open","arguments":[{"name":"book","title":"Book","type":"string"}]},
		{"id":"ledger.sum","title":"Sum"}
	],"slots":`

func readValueSlots(slots string) (manifest.Manifest, error) {
	return manifestjson.Read(strings.NewReader(valuesManifestHead + slots + `}}`))
}

func TestSlotEntryValuesAreDecoded(t *testing.T) {
	parsed, err := readValueSlots(`[
		{"slot":"titleBar.commands","command":"ledger.open","values":{"book":"main"}},
		{"slot":"fileTree.context","command":"ledger.open"},
		{"slot":"palette.searchResult","command":"app:file.open","values":{"placement":"split"}}
	]`)
	if err != nil {
		t.Fatal(err)
	}

	slots := parsed.Contributes.Slots
	if got := slots[0].Values; len(got) != 1 || got["book"] != "main" {
		t.Errorf("the values came out as %v", got)
	}
	if len(slots[1].Values) != 0 {
		t.Errorf("an entry without values came out with %v", slots[1].Values)
	}
	if got := slots[2].Values; got["placement"] != "split" {
		t.Errorf("the values of a command of the application came out as %v", got)
	}
}

func TestASlotEntryWithValuesThatBreakARuleIsAnInvalidSlotEntry(t *testing.T) {
	cases := map[string]string{
		"an argument the command does not declare": `[{"slot":"titleBar.commands","command":"ledger.open","values":{"shelf":"1"}}]`,
		"values for a command without arguments":   `[{"slot":"titleBar.commands","command":"ledger.sum","values":{"book":"main"}}]`,
		"values on a view":                         `[{"slot":"titleBar.status","view":"balance","values":{"book":"main"}}]`,
		"values on a submenu":                      `[{"slot":"fileTree.context","submenu":"ledger.new","title":"New","values":{"book":"main"}}]`,
	}
	for name, slots := range cases {
		_, err := readValueSlots(slots)

		if !errors.Is(err, manifest.ErrInvalidSlotEntry) {
			t.Errorf("%s: error = %v, want an invalid slot entry", name, err)
		}
	}
}

func TestSlotEntryValueAsNumberIsRefused(t *testing.T) {
	_, err := readValueSlots(`[{"slot":"titleBar.commands","command":"ledger.open","values":{"book":1}}]`)

	if err == nil {
		t.Error("no error")
	}
}

func TestAValueForACommandOfTheWindowIsCheckedAgainstTheRegistry(t *testing.T) {
	valid := []string{
		`[{"slot":"titleBar.commands","command":"app:file.open","values":{"placement":"split"}}]`,
		`[{"slot":"titleBar.commands","command":"app:link.import","values":{"address":"https://example.org"}}]`,
		`[{"slot":"titleBar.commands","command":"org.example.harbor:harbor.open","values":{"any":"thing"}}]`,
		`[{"slot":"fileTree.context","command":"app:file.open","values":{"placement":"current"}},{"slot":"fileTree.context","command":"app:file.open","values":{"placement":"split"}}]`,
	}
	for _, slots := range valid {
		if _, err := readValueSlots(slots); err != nil {
			t.Errorf("%s: %v", slots, err)
		}
	}
	invalid := map[string]string{
		"an argument the command does not take":  `[{"slot":"titleBar.commands","command":"app:file.open","values":{"where":"split"}}]`,
		"a value the argument does not allow":    `[{"slot":"titleBar.commands","command":"app:file.open","values":{"placement":"aside"}}]`,
		"values for a command without arguments": `[{"slot":"titleBar.commands","command":"app:settings.open","values":{"placement":"split"}}]`,
		"the same values twice in a slot": `[{"slot":"fileTree.context","command":"app:file.open","values":{"placement":"split"}},` +
			`{"slot":"fileTree.context","command":"app:file.open","values":{"placement":"split"}}]`,
	}
	for name, slots := range invalid {
		if _, err := readValueSlots(slots); !errors.Is(err, manifest.ErrInvalidSlotEntry) {
			t.Errorf("%s: error = %v, want an invalid slot entry", name, err)
		}
	}
}

func TestACommandEntryMayCarryATitleAndAnIcon(t *testing.T) {
	parsed, err := readValueSlots(`[{"slot":"titleBar.commands","command":"ledger.sum","title":"Add up","icon":"calculator"}]`)
	if err != nil {
		t.Fatal(err)
	}

	if one := parsed.Contributes.Slots[0]; one.Title != "Add up" || one.Icon != "calculator" {
		t.Errorf("the entry came out as %+v", one)
	}
	for _, slots := range []string{
		`[{"slot":"titleBar.commands","command":"ledger.sum","title":"  "}]`,
		`[{"slot":"titleBar.commands","command":"ledger.sum","icon":"Calculator"}]`,
	} {
		if _, err := readValueSlots(slots); !errors.Is(err, manifest.ErrInvalidSlotEntry) {
			t.Errorf("%s: error = %v, want an invalid slot entry", slots, err)
		}
	}
}
