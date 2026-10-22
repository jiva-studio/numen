package theme_test

import (
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/theme"
	"github.com/jiva-studio/numen/modules/libs/core/internal/appearance"
)

// fakeAppearances is a service over a themes folder, and the settings behind it, held
// in memory.
type fakeAppearances struct {
	service *theme.Service
	worn    theme.Appearance
	written []theme.Appearance
	said    []string
	refuses error
}

func (d *fakeAppearances) Read() (theme.Appearance, error) { return d.worn, nil }

func (d *fakeAppearances) Write(one theme.Appearance) error {
	if d.refuses != nil {
		return d.refuses
	}
	d.written = append(d.written, one)
	d.worn = one
	return nil
}

func (d *fakeAppearances) Warn(response string) { d.said = append(d.said, response) }

func createFakeAppearances(t *testing.T, worn theme.Appearance) *fakeAppearances {
	t.Helper()
	appearances := &fakeAppearances{worn: worn}
	appearances.service = &theme.Service{
		Catalogue:            createThemeFolder(t),
		Settings:             appearances,
		InterfaceScaleBounds: theme.Bounds{Least: 0.8, Most: 2},
		TextScaleBounds:      theme.Bounds{Least: 0.8, Most: 1.75},
	}
	return appearances
}

func (d *fakeAppearances) themes(t *testing.T) *v1.ListThemesResponse {
	t.Helper()
	answer, err := d.service.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	return answer.Msg
}

func (d *fakeAppearances) choose(t *testing.T, name string, mode v1.Mode) string {
	t.Helper()
	answer, err := d.service.WriteAppearance(t.Context(),
		connect.NewRequest(&v1.WriteAppearanceRequest{Name: name, Mode: mode}))
	if err != nil {
		t.Fatal(err)
	}
	return answer.Msg.GetError()
}

func TestTheListReturnsWhatIsWornAndWhichHalfItIsReadAs(t *testing.T) {
	aged := createFakeAppearances(t, theme.Appearance{ThemeName: "preset:dracula", Mode: appearance.Dark})
	writeTheme(t, aged.service.Catalogue, "mine.css", ":root { --numen-surface: #000000 }")

	answer := aged.themes(t)
	if answer.GetApplied() != "preset:dracula" || answer.GetMode() != v1.Mode_MODE_DARK {
		t.Errorf("wears %q, read as %v", answer.GetApplied(), answer.GetMode())
	}

	origins := map[string]v1.Origin{}
	for _, one := range answer.GetThemes() {
		origins[one.GetName()] = one.GetOrigin()
	}
	if origins["preset:dracula"] != v1.Origin_ORIGIN_PRESET {
		t.Errorf("preset:dracula came off %v", origins["preset:dracula"])
	}
	if origins["mine:mine"] != v1.Origin_ORIGIN_MINE {
		t.Errorf("the person's theme came off %v", origins["mine:mine"])
	}
	for _, one := range answer.GetThemes() {
		if one.GetName() == "preset:dracula" && !one.GetIsPinned() {
			t.Error("a palette published in one half is not offered as pinned")
		}
	}
}

func TestAThemeIsAskedForByNameAndAnyOtherNameIsNothing(t *testing.T) {
	aged := createFakeAppearances(t, theme.Appearance{ThemeName: theme.Default, Mode: appearance.System})
	writeTheme(t, aged.service.Catalogue, "kept.css", ":root { --numen-surface: #123456 }")

	for name, want := range map[string]string{
		"mine:kept":                 "#123456",
		"preset:dracula":            "--numen-surface",
		"mine:../../../.ssh/id_rsa": "",
		"mine:gone":                 "",
	} {
		answer, err := aged.service.ReadTheme(t.Context(),
			connect.NewRequest(&v1.ReadThemeRequest{Name: name}))
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
	answer, err := aged.service.ReadTheme(t.Context(),
		connect.NewRequest(&v1.ReadThemeRequest{Name: theme.Default}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetCss() == "" {
		t.Errorf("%s answered with nothing", theme.Default)
	}
}

func TestChoosingWritesTheThemeAndTheHalfItIsReadAs(t *testing.T) {
	aged := createFakeAppearances(t, theme.Appearance{ThemeName: theme.Default, Mode: appearance.System})
	if failed := aged.choose(t, "preset:nord", v1.Mode_MODE_DARK); failed != "" {
		t.Fatalf("failed: %s", failed)
	}
	if len(aged.written) != 1 || aged.written[0] != (theme.Appearance{ThemeName: "preset:nord", Mode: appearance.Dark}) {
		t.Fatalf("written: %+v", aged.written)
	}
	if answer := aged.themes(t); answer.GetApplied() != "preset:nord" {
		t.Errorf("wears %q", answer.GetApplied())
	}
}

// A client says how far a size goes before a person types a number into it.
func TestTheListReturnsTheTwoSizesAndHowFarEachGoes(t *testing.T) {
	aged := createFakeAppearances(t, theme.Appearance{
		ThemeName:      theme.Default,
		Mode:           appearance.System,
		InterfaceScale: 1.25,
		TextScale:      1.5,
	})

	answer := aged.themes(t)
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
func TestASettingsFileNamingNoSizeIsGeneratedAsDesigned(t *testing.T) {
	aged := createFakeAppearances(t, theme.Appearance{ThemeName: theme.Default, Mode: appearance.System})

	answer := aged.themes(t)
	if answer.GetInterfaceScale() != theme.AsDesigned || answer.GetTextScale() != theme.AsDesigned {
		t.Errorf("drawn at %v and set at %v", answer.GetInterfaceScale(), answer.GetTextScale())
	}
}

// One size is one command, and choosing it says nothing about the other.
func TestChoosingOneSizePreservesTheOtherAsItExists(t *testing.T) {
	aged := createFakeAppearances(t, theme.Appearance{
		ThemeName:      theme.Default,
		Mode:           appearance.System,
		InterfaceScale: 1.25,
		TextScale:      1.5,
	})

	scale := 1.75
	answer, err := aged.service.WriteAppearance(t.Context(), connect.NewRequest(&v1.WriteAppearanceRequest{
		Name:           "preset:nord",
		Mode:           v1.Mode_MODE_DARK,
		InterfaceScale: &scale,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if reason := answer.Msg.GetError(); reason != "" {
		t.Fatalf("failed: %s", reason)
	}
	if len(aged.written) != 1 || aged.written[0].InterfaceScale != 1.75 || aged.written[0].TextScale != 0 {
		t.Fatalf("written: %+v", aged.written)
	}
}

// The settings are left as they are, and the person is standing in front of the
// list that was chosen from.
func TestAChoiceThatCannotBeWornIsRefusedAndReturnsWhy(t *testing.T) {
	aged := createFakeAppearances(t, theme.Appearance{ThemeName: theme.Default, Mode: appearance.System})
	for _, selected := range []struct {
		name string
		mode v1.Mode
	}{
		{"mine:the-one-i-deleted", v1.Mode_MODE_DARK},
		{"mine:../../../.ssh/id_rsa", v1.Mode_MODE_DARK},
		{"preset:nord", v1.Mode_MODE_UNSPECIFIED},
	} {
		failed := aged.choose(t, selected.name, selected.mode)
		if failed == "" {
			t.Errorf("%s, %v was written", selected.name, selected.mode)
		}
		if !strings.Contains(failed, selected.name) {
			t.Errorf("%s failed with %q", selected.name, failed)
		}
	}
	if len(aged.written) != 0 {
		t.Errorf("written: %+v", aged.written)
	}
}

func TestASettingsFileThatWouldNotTakeTheChoiceReturnsSo(t *testing.T) {
	aged := createFakeAppearances(t, theme.Appearance{ThemeName: theme.Default, Mode: appearance.System})
	aged.refuses = errNotWritten
	if failed := aged.choose(t, "preset:nord", v1.Mode_MODE_DARK); !strings.Contains(failed, "read-only") {
		t.Errorf("failed with %q", failed)
	}
}

// A theme deleted out of the folder does not leave the window undressed, and the
func TestAThemeTheSettingsNameThatIsMissingIsSaidAndTheDefaultWorn(t *testing.T) {
	aged := createFakeAppearances(t, theme.Appearance{ThemeName: "mine:the-one-i-deleted", Mode: appearance.Light})
	answer := aged.themes(t)
	if answer.GetApplied() != theme.Default {
		t.Errorf("wears %q", answer.GetApplied())
	}
	if answer.GetMode() != v1.Mode_MODE_LIGHT {
		t.Errorf("read as %v", answer.GetMode())
	}
	if len(aged.said) != 1 || !strings.Contains(aged.said[0], "mine:the-one-i-deleted") {
		t.Errorf("said %v", aged.said)
	}
}

// A build with no settings behind it still lists what it ships.
func TestAServiceGivenNoSettingsWearsThisProductsPalette(t *testing.T) {
	service := &theme.Service{Catalogue: createThemeFolder(t)}
	answer, err := service.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetApplied() != theme.Default || answer.Msg.GetMode() != v1.Mode_MODE_SYSTEM {
		t.Errorf("wears %q, read as %v", answer.Msg.GetApplied(), answer.Msg.GetMode())
	}

	selected, err := service.WriteAppearance(t.Context(),
		connect.NewRequest(&v1.WriteAppearanceRequest{Name: theme.Default, Mode: v1.Mode_MODE_DARK}))
	if err != nil {
		t.Fatal(err)
	}
	if selected.Msg.GetError() == "" {
		t.Error("a build that writes no settings said it had written them")
	}
}

var errNotWritten = errors.New("the settings are read-only")
