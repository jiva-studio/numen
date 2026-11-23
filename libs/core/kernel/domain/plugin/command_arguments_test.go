package plugin_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func getConvert() manifest.Command {
	return manifest.Command{
		ID: "convert",
		Arguments: []manifest.CommandArgument{
			{Name: "pages", Type: manifest.CommandArgumentNumber, IsRequired: true},
			{Name: "again", Type: manifest.CommandArgumentBoolean},
		},
	}
}

func TestCheckCommandArguments_RejectsWhatTheDeclarationDoesNotAllow(t *testing.T) {
	cases := map[string]map[string]string{
		"a required argument left out": {},
		"an argument it does not take": {"pages": "1", "color": "red"},
		"a number that is none":        {"pages": "twelve"},
		"a boolean that is none":       {"pages": "1", "again": "maybe"},
		"a version without items":      {"pages": "1", manifest.CommandArgumentVersion: "a/b"},
	}
	for name, arguments := range cases {
		t.Run(name, func(t *testing.T) {
			err := plugin.CheckCommandArguments(getConvert(), plugin.CommandCall{Arguments: arguments}, false)

			if !errors.Is(err, plugin.ErrBadArguments) {
				t.Errorf("got %v, want ErrBadArguments", err)
			}
		})
	}
}

func TestCheckCommandArguments_AcceptsTheDeclaredArgumentsAndTheVersionOfACommandWithItems(t *testing.T) {
	call := plugin.CommandCall{Arguments: map[string]string{"pages": "12", "again": "1", manifest.CommandArgumentVersion: "a/b"}}

	if err := plugin.CheckCommandArguments(getConvert(), call, true); err != nil {
		t.Errorf("got %v", err)
	}
}

func TestNormalizeCommandArguments_WritesABooleanAsTrueOrFalse(t *testing.T) {
	got := plugin.NormalizeCommandArguments(getConvert(), map[string]string{"pages": "12", "again": "1"})
	if got["again"] != "true" || got["pages"] != "12" {
		t.Errorf("got %v", got)
	}
	got = plugin.NormalizeCommandArguments(getConvert(), map[string]string{"again": "maybe"})
	if got["again"] != "false" {
		t.Errorf("got %v", got)
	}
}
