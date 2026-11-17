package settings_test

import (
	"encoding/json"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	domainsettings "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

func newStoreAt(path string) *settings.Store {
	return settings.NewStore(settings.NewFiles(), func() (string, error) { return path, nil }, nil, nil)
}

func getEditorOf(t *testing.T, content string) map[string]any {
	t.Helper()
	var root struct {
		App struct {
			Editor map[string]any `json:"editor"`
		} `json:"app"`
	}
	if err := json.Unmarshal([]byte(content), &root); err != nil {
		t.Fatal(err)
	}
	return root.App.Editor
}

func TestTheEditorSettingsOfTheFileAreInForce(t *testing.T) {
	file := readFileWith(t, `{"app":{"editor":{"fontSize":19,"fontFamily":"serif","lineHeight":2,"lineNumbers":true,"softWrap":false}}}`)
	want := map[string]any{
		"fontSize": 19.0, "fontFamily": "serif", "lineHeight": 2.0, "lineNumbers": true, "softWrap": false,
	}
	for key, value := range want {
		if got, _ := file.GetValue(domainsettings.AppOwner, "editor", key); got != value {
			t.Errorf("editor.%s in force is %v, the file has %v", key, got, value)
		}
	}
}

func TestTheAnswerToSavingAnEditorSettingCarriesItsNewValue(t *testing.T) {
	path := write(t, `{"v":1,"app":{"appearance":{"mode":"dark"}}}`)
	content, _, err := newStoreAt(path).WriteJSON([]domainsettings.Setting{
		{Path: []string{"app", "editor", "fontSize"}, JSON: "19"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := getEditorOf(t, content)["fontSize"]; got != 19.0 {
		t.Errorf("the answer holds editor.fontSize %v, want 19; the answer is %s", got, content)
	}
}

func TestAReadAfterSavingAnEditorSettingReturnsItsNewValue(t *testing.T) {
	path := write(t, `{"v":1,"app":{"appearance":{"mode":"dark"}}}`)
	store := newStoreAt(path)
	if _, _, err := store.WriteJSON([]domainsettings.Setting{
		{Path: []string{"app", "editor", "lineNumbers"}, JSON: "true"},
	}); err != nil {
		t.Fatal(err)
	}
	content, _, _, err := store.ReadJSON()
	if err != nil {
		t.Fatal(err)
	}
	if got := getEditorOf(t, content)["lineNumbers"]; got != true {
		t.Errorf("the read holds editor.lineNumbers %v, want true; the read is %s", got, content)
	}
}

func TestTheEditorSettingsOfTheFileAreNotLostByASavedEdit(t *testing.T) {
	body := `{"v":1,"app":{"editor":{"fontSize":19},"appearance":{"mode":"dark"}}}`
	path := write(t, body)
	content, _, err := newStoreAt(path).WriteJSON([]domainsettings.Setting{
		{Path: []string{"app", "appearance", "mode"}, JSON: `"light"`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := getEditorOf(t, content)["fontSize"]; got != 19.0 {
		t.Errorf("the answer holds editor.fontSize %v, want 19 as in the file; the answer is %s", got, content)
	}
}
