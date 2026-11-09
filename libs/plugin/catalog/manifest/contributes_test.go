package manifest_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const contributesHead = `{"id":"a.b","name":"A","version":"1.0.0","frontend":{}`

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

func TestATabKindDeclaresTheNotesItOpensAndItsRank(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(contributesHead + `,"contributes":{"tabKinds":[
		{"kind":"deck","title":"Deck","when":"target.noteType == 'deck'"},
		{"kind":"note","title":"Note","when":"target.isNote","rank":"alternate"}]}}`))
	if err != nil {
		t.Fatal(err)
	}

	deck, _ := m.Contributes.GetTabKind("deck")
	note, _ := m.Contributes.GetTabKind("note")
	if deck.When != "target.noteType == 'deck'" || deck.Rank != domain.RankDefault {
		t.Errorf("the deck kind = %+v, want its condition and the default rank", deck)
	}
	if note.When != "target.isNote" || note.Rank != domain.RankAlternate {
		t.Errorf("the note kind = %+v, want its condition and the alternate rank", note)
	}
}

func TestAContributionThatTheApplicationCannotUseIsRefused(t *testing.T) {
	cases := map[string]string{
		"a kind in capitals":    `{"tabKinds":[{"kind":"Plex","title":"P"}]}`,
		"a kind with no name":   `{"tabKinds":[{"kind":"","title":"P"}]}`,
		"a kind declared twice": `{"tabKinds":[{"kind":"plex","title":"P"},{"kind":"plex","title":"Q"}]}`,
		"a kind with no title":  `{"tabKinds":[{"kind":"plex","title":" "}]}`,
		"an icon in capitals":   `{"tabKinds":[{"kind":"plex","title":"P","icon":"Waypoints"}]}`,
		"an unreadable when":    `{"tabKinds":[{"kind":"plex","title":"P","when":"target.isRemote"}]}`,
		"an unknown rank":       `{"tabKinds":[{"kind":"plex","title":"P","rank":"first"}]}`,
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

func TestCheckThemeFileAndThemeIssues(t *testing.T) {
	if manifest.MaxThemeSize != 256<<10 {
		t.Fatalf("MaxThemeSize = %d, want %d", manifest.MaxThemeSize, 256<<10)
	}
	if err := manifest.CheckThemeFile("themes/night.css"); err != nil {
		t.Errorf("CheckThemeFile valid: %v", err)
	}
	if err := manifest.CheckThemeFile("a\x01n.css"); err != nil {
		t.Errorf("CheckThemeFile with rune 1: %v", err)
	}
	for name, tc := range map[string]struct {
		file string
		want string
	}{
		"empty":          {"", "file is empty"},
		"backslash":      {`a\n.css`, "not a slash-separated path"},
		"null byte":      {"a\x00n.css", "not a slash-separated path"},
		"absolute":       {"/n.css", "not a clean relative path"},
		"unclean":        {"a//n.css", "not a clean relative path"},
		"dotdot":         {"..", "leaves the plugin folder"},
		"dotdot prefix":  {"../n.css", "leaves the plugin folder"},
		"wrong ext":      {"n.txt", "not a .css stylesheet"},
		"bare extension": {".css", "not a .css stylesheet"},
		"dir bare ext":   {"a/.css", "not a .css stylesheet"},
	} {
		err := manifest.CheckThemeFile(tc.file)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: CheckThemeFile(%q) = %v, want %q", name, tc.file, err, tc.want)
		}
	}

	for name, themes := range map[string]string{
		"capital id":         `[{"id":"Night","name":"N","file":"n.css"}]`,
		"duplicate id":       `[{"id":"n","name":"N","file":"n.css"},{"id":"n","name":"M","file":"m.css"}]`,
		"blank name":         `[{"id":"n","name":" ","file":"n.css"}]`,
		"invalid appearance": `[{"id":"n","name":"N","appearance":"sepia","file":"n.css"}]`,
		"invalid file":       `[{"id":"n","name":"N","file":"../n.css"}]`,
	} {
		_, err := manifestjson.Read(strings.NewReader(contributesHead + `,"contributes":{"themes":` + themes + `}}`))
		if !errors.Is(err, manifest.ErrInvalidContribution) {
			t.Errorf("%s: error = %v, want ErrInvalidContribution", name, err)
		}
	}
}
