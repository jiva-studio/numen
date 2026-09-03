package theme_test

import (
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/theme"
)

// dressing is a service over a themes folder, and the settings behind it, held
// in memory.
type dressing struct {
	service *theme.Service
	worn    theme.Appearance
	written []theme.Appearance
	said    []string
	refuses error
}

func (d *dressing) Read() (theme.Appearance, error) { return d.worn, nil }

func (d *dressing) Write(one theme.Appearance) error {
	if d.refuses != nil {
		return d.refuses
	}
	d.written = append(d.written, one)
	d.worn = one
	return nil
}

func (d *dressing) Warn(said string) { d.said = append(d.said, said) }

func dressed(t *testing.T, worn theme.Appearance) *dressing {
	t.Helper()
	kept := &dressing{worn: worn}
	kept.service = &theme.Service{
		Catalogue:            folder(t),
		Settings:             kept,
		InterfaceScaleBounds: theme.Bounds{Least: 0.8, Most: 2},
		TextScaleBounds:      theme.Bounds{Least: 0.8, Most: 1.75},
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
	worn := dressed(t, theme.Appearance{ThemeName: "preset:dracula", Mode: v1.Mode_MODE_DARK})
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
	worn := dressed(t, theme.Appearance{ThemeName: theme.Default, Mode: v1.Mode_MODE_SYSTEM})
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
	worn := dressed(t, theme.Appearance{ThemeName: theme.Default, Mode: v1.Mode_MODE_SYSTEM})
	if failed := worn.choose(t, "preset:nord", v1.Mode_MODE_DARK); failed != "" {
		t.Fatalf("refused: %s", failed)
	}
	if len(worn.written) != 1 || worn.written[0] != (theme.Appearance{ThemeName: "preset:nord", Mode: v1.Mode_MODE_DARK}) {
		t.Fatalf("written: %+v", worn.written)
	}
	if answer := worn.themes(t); answer.GetApplied() != "preset:nord" {
		t.Errorf("wears %q", answer.GetApplied())
	}
}

// A client says how far a size goes before a person types a number into it.
func TestTheListSaysTheTwoSizesAndHowFarEachGoes(t *testing.T) {
	worn := dressed(t, theme.Appearance{
		ThemeName:      theme.Default,
		Mode:           v1.Mode_MODE_SYSTEM,
		InterfaceScale: 1.25,
		TextScale:      1.5,
	})

	answer := worn.themes(t)
	if answer.GetInterfaceScale() != 1.25 || answer.GetTextScale() != 1.5 {
		t.Errorf("drawn at %v and set at %v", answer.GetInterfaceScale(), answer.GetTextScale())
	}
	if answer.GetInterfaceScaleBounds().GetLeast() != 0.8 || answer.GetInterfaceScaleBounds().GetMost() != 2 {
		t.Errorf("the interface goes %v", answer.GetInterfaceScaleBounds())
	}
	if answer.GetTextScaleBounds().GetLeast() != 0.8 || answer.GetTextScaleBounds().GetMost() != 1.75 {
		t.Errorf("the text goes %v", answer.GetTextScaleBounds())
	}
}

// A window that has been dressed by nothing is drawn at the size it was
// designed at.
func TestASettingsFileNamingNoSizeIsDrawnAsDesigned(t *testing.T) {
	worn := dressed(t, theme.Appearance{ThemeName: theme.Default, Mode: v1.Mode_MODE_SYSTEM})

	answer := worn.themes(t)
	if answer.GetInterfaceScale() != theme.AsDesigned || answer.GetTextScale() != theme.AsDesigned {
		t.Errorf("drawn at %v and set at %v", answer.GetInterfaceScale(), answer.GetTextScale())
	}
}

// One size is one command, and choosing it says nothing about the other.
func TestChoosingOneSizeLeavesTheOtherAsItStands(t *testing.T) {
	worn := dressed(t, theme.Appearance{
		ThemeName:      theme.Default,
		Mode:           v1.Mode_MODE_SYSTEM,
		InterfaceScale: 1.25,
		TextScale:      1.5,
	})

	drawn := 1.75
	answer, err := worn.service.Choose(t.Context(), connect.NewRequest(&v1.ChooseRequest{
		Name:           "preset:nord",
		Mode:           v1.Mode_MODE_DARK,
		InterfaceScale: &drawn,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if failed := answer.Msg.GetFailed(); failed != "" {
		t.Fatalf("refused: %s", failed)
	}
	if len(worn.written) != 1 || worn.written[0].InterfaceScale != 1.75 || worn.written[0].TextScale != 0 {
		t.Fatalf("written: %+v", worn.written)
	}
}

// The settings are left as they are, and the person is standing in front of the
// list that was chosen from.
func TestAChoiceThatCannotBeWornIsRefusedAndSaysWhy(t *testing.T) {
	worn := dressed(t, theme.Appearance{ThemeName: theme.Default, Mode: v1.Mode_MODE_SYSTEM})
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
	worn := dressed(t, theme.Appearance{ThemeName: theme.Default, Mode: v1.Mode_MODE_SYSTEM})
	worn.refuses = errNotWritten
	if failed := worn.choose(t, "preset:nord", v1.Mode_MODE_DARK); !strings.Contains(failed, "read-only") {
		t.Errorf("refused with %q", failed)
	}
}

// A theme deleted out of the folder does not leave the window undressed, and the
// name it was under is what the person is told.
func TestAThemeTheSettingsNameThatIsGoneIsSaidAndTheDefaultWorn(t *testing.T) {
	worn := dressed(t, theme.Appearance{ThemeName: "mine:the-one-i-deleted", Mode: v1.Mode_MODE_LIGHT})
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
