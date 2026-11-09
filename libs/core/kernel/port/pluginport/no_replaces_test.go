package pluginport_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

func TestTheSettingAWindowDrawsCarriesNoOldPlaceOfIt(t *testing.T) {
	if _, found := reflect.TypeFor[pluginport.PluginSetting]().FieldByName("Replaces"); found {
		t.Error("PluginSetting has the field Replaces")
	}
}
