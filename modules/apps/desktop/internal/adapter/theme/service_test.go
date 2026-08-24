package theme_test

import (
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/theme"
)

// dressing is a service over a themes folder, with the settings behind it as
// the two functions it is given.
type dressing struct {
	service *theme.Service
	worn    theme.Dress
	written []theme.Dress
	said    []string
	refuses error
}

func dressed(t *testing.T, worn theme.Dress) *dressing {
	t.Helper()
	kept := &dressing{worn: worn}
	kept.service = &theme.Service{
		Catalogue: folder(t),
		Dressed:   func() (theme.Dress, error) { return kept.worn, nil },
		Wear: func(one theme.Dress) error {
			if kept.refuses != nil {
				return kept.refuses
			}
			kept.written = append(kept.written, one)
			kept.worn = one
			return nil
		},
		Say: func(said string) { kept.said = append(kept.said, said) },
	}
	return kept
}

func (d *dressing) themes(t *testing.T) *v1.ThemesResponse {
	t.Helper()
	answer, err := d.service.Themes(t.Context(), connect.NewRequest(&v1.ThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	return answer.Msg
}

func (d *dressing) choose(t *testing.T, name string, mode v1.Mode) string {
	t.Helper()
	answer, err := d.service.Choose(t.Context(),
		connect.NewRequest(&v1.ChooseRequest{Name: name, Mode: mode}))
	if err != nil {
		t.Fatal(err)
	}
	return answer.Msg.GetFailed()
}

func TestTheListSaysWhatIsWornAndWhichHalfItIsReadAs(t *testing.T) {
	worn := dressed(t, theme.Dress{Theme: "preset:dracula", Mode: v1.Mode_MODE_DARK})
	put(t, worn.service.Catalogue, "mine.css", ":root { --numen-surface: #000000 }")

	answer := worn.themes(t)
	if answer.GetApplied() != "preset:dracula" || answer.GetMode() != v1.Mode_MODE_DARK {
		t.Errorf("wears %q, read as %v", answer.GetApplied(), answer.GetMode())
	}

	shelves := map[string]v1.Shelf{}
	for _, one := range answer.GetThemes() {
		shelves[one.GetName()] = one.GetShelf()
	}
	if shelves["preset:dracula"] != v1.Shelf_SHELF_PRESET {
		t.Errorf("preset:dracula came off %v", shelves["preset:dracula"])
	}
	if shelves["mine:mine"] != v1.Shelf_SHELF_MINE {
		t.Errorf("the person's theme came off %v", shelves["mine:mine"])
	}
	for _, one := range answer.GetThemes() {
		if one.GetName() == "preset:dracula" && !one.GetPinned() {
			t.Error("a palette published in one half is not offered as pinned")
		}
	}
}

func TestAThemeIsAskedForByNameAndAnyOtherNameIsNothing(t *testing.T) {
	worn := dressed(t, theme.Dress{Theme: theme.Default, Mode: v1.Mode_MODE_SYSTEM})
	put(t, worn.service.Catalogue, "kept.css", ":root { --numen-surface: #123456 }")

	for name, want := range map[string]string{
		"mine:kept":                 "#123456",
		"preset:dracula":            "--numen-surface",
		"mine:../../../.ssh/id_rsa": "",
		"mine:gone":                 "",
	} {
		answer, err := worn.service.Theme(t.Context(),
			connect.NewRequest(&v1.ThemeRequest{Name: name}))
		if err != nil {
			t.Fatal(err)
		}
		if want == "" && answer.Msg.GetCss() != "" {
			t.Errorf("%s answered with %q", name, answer.Msg.GetCss())
		}
		if want != "" && !strings.Contains(answer.Msg.GetCss(), want) {
			t.Errorf("%s answered with %q", name, answer.Msg.GetCss())
		}
	}

	// This product's own palette names no token: what `tokens.css` holds is
	// what it is. It answers with its file all the same.
	answer, err := worn.service.Theme(t.Context(),
		connect.NewRequest(&v1.ThemeRequest{Name: theme.Default}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetCss() == "" {
		t.Errorf("%s answered with nothing", theme.Default)
	}
}

func TestChoosingWritesTheThemeAndTheHalfItIsReadAs(t *testing.T) {
	worn := dressed(t, theme.Dress{Theme: theme.Default, Mode: v1.Mode_MODE_SYSTEM})
	if failed := worn.choose(t, "preset:nord", v1.Mode_MODE_DARK); failed != "" {
		t.Fatalf("refused: %s", failed)
	}
	if len(worn.written) != 1 || worn.written[0] != (theme.Dress{Theme: "preset:nord", Mode: v1.Mode_MODE_DARK}) {
		t.Fatalf("written: %+v", worn.written)
	}
	if answer := worn.themes(t); answer.GetApplied() != "preset:nord" {
		t.Errorf("wears %q", answer.GetApplied())
	}
}

// The settings are left as they are, and the person is standing in front of the
// list that was chosen from.
func TestAChoiceThatCannotBeWornIsRefusedAndSaysWhy(t *testing.T) {
	worn := dressed(t, theme.Dress{Theme: theme.Default, Mode: v1.Mode_MODE_SYSTEM})
	for _, chosen := range []struct {
		name string
		mode v1.Mode
	}{
		{"mine:the-one-i-deleted", v1.Mode_MODE_DARK},
		{"mine:../../../.ssh/id_rsa", v1.Mode_MODE_DARK},
		{"preset:nord", v1.Mode_MODE_UNSPECIFIED},
	} {
		failed := worn.choose(t, chosen.name, chosen.mode)
		if failed == "" {
			t.Errorf("%s, %v was written", chosen.name, chosen.mode)
		}
		if !strings.Contains(failed, chosen.name) {
			t.Errorf("%s was refused with %q", chosen.name, failed)
		}
	}
	if len(worn.written) != 0 {
		t.Errorf("written: %+v", worn.written)
	}
}

func TestASettingsFileThatWouldNotTakeTheChoiceSaysSo(t *testing.T) {
	worn := dressed(t, theme.Dress{Theme: theme.Default, Mode: v1.Mode_MODE_SYSTEM})
	worn.refuses = errNotWritten
	if failed := worn.choose(t, "preset:nord", v1.Mode_MODE_DARK); !strings.Contains(failed, "read-only") {
		t.Errorf("refused with %q", failed)
	}
}

// A theme deleted out of the folder does not leave the window undressed, and the
// name it was under is what the person is told.
func TestAThemeTheSettingsNameThatIsGoneIsSaidAndTheDefaultWorn(t *testing.T) {
	worn := dressed(t, theme.Dress{Theme: "mine:the-one-i-deleted", Mode: v1.Mode_MODE_LIGHT})
	answer := worn.themes(t)
	if answer.GetApplied() != theme.Default {
		t.Errorf("wears %q", answer.GetApplied())
	}
	if answer.GetMode() != v1.Mode_MODE_LIGHT {
		t.Errorf("read as %v", answer.GetMode())
	}
	if len(worn.said) != 1 || !strings.Contains(worn.said[0], "mine:the-one-i-deleted") {
		t.Errorf("said %v", worn.said)
	}
}

// A build with no settings behind it still lists what it ships.
func TestAServiceGivenNoSettingsWearsThisProductsPalette(t *testing.T) {
	service := &theme.Service{Catalogue: folder(t)}
	answer, err := service.Themes(t.Context(), connect.NewRequest(&v1.ThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetApplied() != theme.Default || answer.Msg.GetMode() != v1.Mode_MODE_SYSTEM {
		t.Errorf("wears %q, read as %v", answer.Msg.GetApplied(), answer.Msg.GetMode())
	}

	chosen, err := service.Choose(t.Context(),
		connect.NewRequest(&v1.ChooseRequest{Name: theme.Default, Mode: v1.Mode_MODE_DARK}))
	if err != nil {
		t.Fatal(err)
	}
	if chosen.Msg.GetFailed() == "" {
		t.Error("a build that writes no settings said it had written them")
	}
}

var errNotWritten = errors.New("the settings are read-only")
