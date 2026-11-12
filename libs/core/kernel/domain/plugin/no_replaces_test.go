package plugin_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func TestTheSettingAWindowDrawsCarriesNoOldPlaceOfIt(t *testing.T) {
	if _, found := reflect.TypeFor[plugin.Setting]().FieldByName("Replaces"); found {
		t.Error("Setting has the field Replaces")
	}
}
