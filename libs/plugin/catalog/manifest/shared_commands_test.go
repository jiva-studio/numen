package manifest_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestASharedCommandDeclaresItsArgumentsOnce(t *testing.T) {
	want := []manifest.CommandArgument{{Name: "address", Title: "Web address", Type: manifest.CommandArgumentString, IsRequired: true}}
	if got := manifest.GetSharedCommandArguments("app:link.import"); !reflect.DeepEqual(got, want) {
		t.Errorf("app:link.import takes %+v, want %+v", got, want)
	}
	if got := manifest.GetSharedCommandArguments("app:file.createText"); len(got) != 0 {
		t.Errorf("app:file.createText takes %+v, want none", got)
	}
}
