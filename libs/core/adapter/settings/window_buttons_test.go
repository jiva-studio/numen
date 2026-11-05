package settings_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
)

// The declaration cannot say that a setting applies on Linux only: a condition holds while another
// setting has one of the values listed, and a platform is not a setting. The window ignores the
// setting on Windows and macOS.
func TestTheWindowButtonsAreAnEnumOfTheStylesTheTitleBarDraws(t *testing.T) {
	declared := findDeclared(t, "window.buttons")

	if declared.Type != "enum" {
		t.Errorf("the window buttons are %s", declared.Type)
	}
	var got []string
	for _, option := range declared.Options {
		got = append(got, option.Value)
	}
	if want := []string{"windows", "mac"}; !slices.Equal(got, want) {
		t.Errorf("the window buttons offer %v, want %v", got, want)
	}
	for _, option := range declared.Options {
		if option.Title == "" {
			t.Errorf("the option %s has no title", option.Value)
		}
	}
}

func TestTheWindowButtonsStandAtTheRightByDefault(t *testing.T) {
	if got := findDeclared(t, "window.buttons").Default; got != "windows" {
		t.Errorf("the window buttons default to %v", got)
	}
}

func TestTheWindowButtonsAreTitledInPlainWords(t *testing.T) {
	declared := findDeclared(t, "window.buttons")

	if declared.Title != "Window buttons" {
		t.Errorf("the window buttons are titled %q", declared.Title)
	}
	if declared.Description == "" {
		t.Error("the window buttons have no description")
	}
}

func readFileWith(t *testing.T, body string) settings.File {
	t.Helper()
	file := settings.DefaultFile()
	if err := json.Unmarshal([]byte(body), &file); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestTheWindowButtonsTheFileHoldsAreInForce(t *testing.T) {
	file := readFileWith(t, `{"app":{"window":{"buttons":"mac"}}}`)

	got, found := file.GetValue("app", "window", "buttons")

	if !found || got != "mac" {
		t.Errorf("the window buttons in force are %v (found %v), the file holds mac", got, found)
	}
}

func TestTheWindowButtonsStandAtTheRightInAFileThatLeavesThemOut(t *testing.T) {
	file := readFileWith(t, `{"app":{"appearance":{"mode":"dark"}}}`)

	got, found := file.GetValue("app", "window", "buttons")

	if !found || got != "windows" {
		t.Errorf("the window buttons in force are %v (found %v), want windows", got, found)
	}
}

func TestTheWindowButtonsAreWrittenBackWithTheSettings(t *testing.T) {
	file := readFileWith(t, `{"app":{"window":{"buttons":"mac"}}}`)

	written, err := file.WriteJSON()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(written, `"window":{"buttons":"mac"}`) {
		t.Errorf("the settings written back are %s, without the window buttons", written)
	}
}

// Every setting the shell declares is read from the file into force, not only the ones whose
// section the document has a field for.
func TestEverySettingOfTheShellTheFileHoldsIsInForce(t *testing.T) {
	file := readFileWith(t, `{"app":{
		"appearance":{"theme":"mine:sea","mode":"dark","interfaceScale":1.25,"textScale":1.5},
		"naming":{"syncTitleAndFilename":false},
		"window":{"buttons":"mac"},
		"performance":{"profile":"eco"}}}`)
	want := map[string]any{
		"appearance.theme":            "mine:sea",
		"appearance.mode":             "dark",
		"appearance.interfaceScale":   1.25,
		"appearance.textScale":        1.5,
		"naming.syncTitleAndFilename": false,
		"window.buttons":              "mac",
		"performance.profile":         "eco",
	}
	for _, declared := range settings.GetAppDeclaration() {
		path := append([]string{settings.OwnerApp}, strings.Split(declared.Key, ".")...)
		if got, _ := file.GetValue(path...); got != want[declared.Key] {
			t.Errorf("%s in force is %v, the file holds %v", declared.Key, got, want[declared.Key])
		}
	}
}
