package settings_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	domainsettings "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

func readFileWith(t *testing.T, body string) settings.File {
	t.Helper()
	file := settings.DefaultFile()
	if err := json.Unmarshal([]byte(body), &file); err != nil {
		t.Fatal(err)
	}
	return file
}

// Every setting the shell declares is read from the file into force, not only the ones whose
// section the document has a field for.
func TestEverySettingOfTheShellInTheFileIsInForce(t *testing.T) {
	file := readFileWith(t, `{"app":{
		"appearance":{"theme":"mine:sea","mode":"dark","interfaceScale":1.25,"textScale":1.5},
		"naming":{"syncTitleAndFilename":false},
		"performance":{"profile":"eco"}}}`)
	want := map[string]any{
		"appearance.theme":            "mine:sea",
		"appearance.mode":             "dark",
		"appearance.interfaceScale":   1.25,
		"appearance.textScale":        1.5,
		"naming.syncTitleAndFilename": false,
		"performance.profile":         "eco",
	}
	for _, declared := range settings.GetAppDeclaration() {
		path := append([]string{domainsettings.AppOwner}, strings.Split(declared.Key, ".")...)
		if got, _ := file.GetValue(path...); got != want[declared.Key] {
			t.Errorf("%s in force is %v, the file has %v", declared.Key, got, want[declared.Key])
		}
	}
}
