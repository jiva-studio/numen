package settings_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
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
		"editor":{"fontSize":19,"fontFamily":"serif","lineHeight":2,"lineNumbers":true,"highlightActiveLine":true,"softWrap":false},
		"performance":{"profile":"eco","disableDmabufRenderer":false},
		"search":{"rerankWindow":70,"maxReranked":200,"maxRerankedBytes":2048,"defaultLimit":30,"maxLimit":90,
			"candidatesPerCondition":250,"maxCandidates":900,"maxPathPrefixes":4,"mergeGapBytes":100,"maxQueryBytes":8192},
		"logging":{"level":"debug"},
		"keys":{"credentialsFile":"passphrase"}}}`)
	want := map[string]any{
		"appearance.theme":                  "mine:sea",
		"appearance.mode":                   "dark",
		"appearance.interfaceScale":         1.25,
		"appearance.textScale":              1.5,
		"naming.syncTitleAndFilename":       false,
		"editor.fontSize":                   19.0,
		"editor.fontFamily":                 "serif",
		"editor.lineHeight":                 2.0,
		"editor.lineNumbers":                true,
		"editor.highlightActiveLine":        true,
		"editor.softWrap":                   false,
		"performance.profile":               "eco",
		"performance.disableDmabufRenderer": false,
		"search.rerankWindow":               70.0,
		"search.maxReranked":                200.0,
		"search.maxRerankedBytes":           2048.0,
		"search.defaultLimit":               30.0,
		"search.maxLimit":                   90.0,
		"search.candidatesPerCondition":     250.0,
		"search.maxCandidates":              900.0,
		"search.maxPathPrefixes":            4.0,
		"search.mergeGapBytes":              100.0,
		"search.maxQueryBytes":              8192.0,
		"logging.level":                     "debug",
		"keys.credentialsFile":              "passphrase",
	}
	for _, declared := range settings.GetAppDeclaration() {
		path := append([]string{domainsettings.AppOwner}, strings.Split(declared.Key, ".")...)
		if got, _ := file.GetValue(path...); got != want[declared.Key] {
			t.Errorf("%s in force is %v, the file has %v", declared.Key, got, want[declared.Key])
		}
	}
}

func TestLogLevelReadsApplicationLoggingSetting(t *testing.T) {
	defaultFile := settings.DefaultFile()
	if got := defaultFile.LogLevel(); got != logs.DefaultLevel {
		t.Fatalf("default LogLevel = %v, want %v", got, logs.DefaultLevel)
	}

	debug := readFileWith(t, `{"app":{"logging":{"level":"debug"},
		"keys":{"credentialsFile":"passphrase"}}}`)
	if got := debug.LogLevel(); got != logs.LevelDebug {
		t.Fatalf("debug LogLevel = %v, want %v", got, logs.LevelDebug)
	}
}
