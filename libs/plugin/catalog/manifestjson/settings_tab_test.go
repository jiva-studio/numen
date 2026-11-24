package manifestjson_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const tabsManifest = `{"id":"a.b","name":"A","version":"1.0.0","frontend":{},%s}`

func readTabs(t *testing.T, fields string) (manifest.Manifest, error) {
	t.Helper()
	return manifestjson.Read(strings.NewReader(strings.Replace(tabsManifest, "%s", fields, 1)))
}

func TestManifestReadsTheTabsOfTheSettingsPageAndTheTabOfAView(t *testing.T) {
	m, err := readTabs(t, `"settingsTabs":[{"id":"sync","title":"Sync"}],
		"settingsViews":[{"id":"vaults","title":"Vaults","tab":"sync"},{"id":"plain","title":"Plain"},{"id":"main","title":"Main","tab":"settings"}]`)
	if err != nil {
		t.Fatal(err)
	}

	if len(m.SettingsTabs) != 1 || m.SettingsTabs[0] != (manifest.SettingsTab{ID: "sync", Title: "Sync"}) {
		t.Errorf("SettingsTabs = %+v, want the tab sync titled Sync", m.SettingsTabs)
	}
	if m.SettingsViews[0].Tab != "sync" || m.SettingsViews[1].Tab != "" || m.SettingsViews[2].Tab != "settings" {
		t.Errorf("SettingsViews = %+v, want the tabs sync, none and settings", m.SettingsViews)
	}
}

func TestManifestRejectsASettingsTabDeclaredWrongly(t *testing.T) {
	tests := map[string]string{
		"no id":         `"settingsTabs":[{"title":"Sync"}]`,
		"no title":      `"settingsTabs":[{"id":"sync"}]`,
		"twice":         `"settingsTabs":[{"id":"sync","title":"Sync"},{"id":"sync","title":"Again"}]`,
		"the default":   `"settingsTabs":[{"id":"settings","title":"Settings"}]`,
		"view of a tab": `"settingsViews":[{"id":"vaults","title":"Vaults","tab":"sync"}]`,
	}
	for name, fields := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := readTabs(t, fields)

			if !errors.Is(err, manifest.ErrInvalidSettingsView) {
				t.Fatalf("error = %v, want ErrInvalidSettingsView", err)
			}
		})
	}
}
