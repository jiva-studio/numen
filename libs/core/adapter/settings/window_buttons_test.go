package settings_test

import (
	"slices"
	"testing"
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
