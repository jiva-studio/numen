package manifest_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

const contributesHead = `{"id":"a.b","name":"A","version":"1.0.0","frontend":{"entrypoint":"dist/index.js"}`

func TestAManifestDeclaresTheTabsFeatureAndCategoryOfThePlugin(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(contributesHead + `,"contributes":{
		"tabKinds":[{"kind":"plex","title":"Plex","icon":"waypoints"},{"kind":"reader:media-player","title":"Player"}],
		"feature":"plex graph","category":"views"}}`))
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(m.Contributes.ListTabKinds(), []string{"plex", "reader:media-player"}) {
		t.Errorf("tab kinds = %v", m.Contributes.ListTabKinds())
	}
	kind, ok := m.Contributes.GetTabKind("plex")
	if !ok || kind.Title != "Plex" || kind.Icon != "waypoints" {
		t.Errorf("the plex kind = %+v, %v", kind, ok)
	}
	if m.Contributes.HasTabKind("deck") || m.Contributes.Feature != "plex graph" || m.Contributes.Category != "views" {
		t.Errorf("contributes = %+v", m.Contributes)
	}
}

func TestAContributionThatTheApplicationCannotUseIsRefused(t *testing.T) {
	cases := map[string]string{
		"a kind in capitals":    `{"tabKinds":[{"kind":"Plex","title":"P"}]}`,
		"a kind with no name":   `{"tabKinds":[{"kind":"","title":"P"}]}`,
		"a kind declared twice": `{"tabKinds":[{"kind":"plex","title":"P"},{"kind":"plex","title":"Q"}]}`,
		"a kind with no title":  `{"tabKinds":[{"kind":"plex","title":" "}]}`,
		"an icon in capitals":   `{"tabKinds":[{"kind":"plex","title":"P","icon":"Waypoints"}]}`,
		"a category with space": `{"category":"my views"}`,
		"a padded feature":      `{"feature":" plex "}`,
	}
	for name, contributes := range cases {
		_, err := manifestjson.Read(strings.NewReader(contributesHead + `,"contributes":` + contributes + `}`))

		if !errors.Is(err, manifest.ErrInvalidContribution) {
			t.Errorf("%s: error = %v, want an invalid contribution", name, err)
		}
	}
}

func TestATabNeedsAFrontendToRegisterIt(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(
		`{"id":"a.b","name":"A","version":"1.0.0","contributes":{"tabKinds":[{"kind":"plex","title":"P"}]}}`))

	if !errors.Is(err, manifest.ErrInvalidContribution) || !strings.Contains(err.Error(), "no frontend") {
		t.Errorf("error = %v, want a refusal for the missing frontend", err)
	}
}
