package manifest_test

import (
	"reflect"
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestNoTypeOfASettingCarriesAnOldPlaceOfIt(t *testing.T) {
	for _, one := range []reflect.Type{
		reflect.TypeFor[manifest.Setting](),
		reflect.TypeFor[v1.ManifestSetting](),
	} {
		if _, found := one.FieldByName("Replaces"); found {
			t.Errorf("%s has the field Replaces", one)
		}
	}
}
