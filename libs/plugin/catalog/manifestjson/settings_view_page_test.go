package manifestjson_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func TestManifestReadsThePageOfASettingsView(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","frontend":{},
		"settingsViews":[{"id":"vaults","title":"Vaults"},{"id":"subscription","title":"Subscription","page":"identity"}]}`))
	if err != nil {
		t.Fatal(err)
	}

	if len(m.SettingsViews) != 2 || m.SettingsViews[0].Page != "" || m.SettingsViews[1].Page != "identity" {
		t.Errorf("SettingsViews = %+v, want the first with no page and the second on the page identity", m.SettingsViews)
	}
}

func TestManifestRejectsAPageOfASettingsViewThatIsNotKnown(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","frontend":{},
		"settingsViews":[{"id":"subscription","title":"Subscription","page":"billing"}]}`))

	if !errors.Is(err, manifest.ErrInvalidSettingsView) {
		t.Fatalf("error = %v, want ErrInvalidSettingsView", err)
	}
	if !strings.Contains(err.Error(), "page") || !strings.Contains(err.Error(), `"billing"`) {
		t.Errorf("error = %q, want it to name the page and its value", err)
	}
}

func TestManifestListIssuesAcceptsTheEmptyPageAndTheIdentityPage(t *testing.T) {
	m := manifest.Manifest{
		ID: "a.b", Name: "A", Version: "1.0.0", Frontend: &manifest.Frontend{},
		SettingsViews: []manifest.SettingsView{{ID: "v", Title: "V", Page: ""}, {ID: "w", Title: "W", Page: "identity"}},
	}

	if issues := m.ListIssues(); len(issues) != 0 {
		t.Errorf("ListIssues = %v, want none", issues)
	}
}
