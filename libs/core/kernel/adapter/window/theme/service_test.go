package theme_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/theme"
	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/themefiles"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/watchhub"
	themeusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/theme"
)

var errNotWritten = errors.New("the settings are read-only")

// fakeAppearances is the settings behind the themes, held in memory.
type fakeAppearances struct {
	appearance themedomain.Appearance
	saves      []themedomain.Appearance
	failure    error
}

func (d *fakeAppearances) ReadAppearance() (themedomain.Appearance, error) { return d.appearance, nil }

func (d *fakeAppearances) WriteAppearance(one themedomain.Appearance) error {
	if d.failure != nil {
		return d.failure
	}
	d.saves = append(d.saves, one)
	d.appearance = one
	return nil
}

func (d *fakeAppearances) WarnTheme(string) {}

func createService(t *testing.T, stored themedomain.Appearance) (*theme.Service, *fakeAppearances, themefiles.Files) {
	t.Helper()
	files, err := themefiles.OpenAt(filepath.Join(t.TempDir(), "numen", "themes"), watchhub.NewHub())
	if err != nil {
		t.Fatal(err)
	}
	settings := &fakeAppearances{appearance: stored, saves: nil, failure: nil}
	sources := themeusecase.Sources{Presets: files, Own: files, Plugins: nil}
	scales := themeusecase.NewScales(0, 0)
	service := theme.NewService(
		themeusecase.NewGetThemes(sources, settings, settings, scales,
			themeusecase.Bounds{Min: 0.8, Max: 2}, themeusecase.Bounds{Min: 0.8, Max: 1.75}),
		themeusecase.NewReadTheme(sources),
		themeusecase.NewWriteAppearance(sources, settings, settings, scales),
		themeusecase.NewWatchThemes(sources, files, 0),
	)
	return service, settings, files
}

func TestTheListIsAnsweredWithTheOriginsTheModeAndTheBounds(t *testing.T) {
	service, _, _ := createService(t, themedomain.Appearance{
		ThemeName: themedomain.Default, Mode: themedomain.Dark, InterfaceScale: 1.25, TextScale: 1.5,
	})

	answer, err := service.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := answer.Msg; got.GetApplied() != themedomain.Default || got.GetMode() != v1.Mode_MODE_DARK ||
		got.GetInterfaceScale() != 1.25 || got.GetTextScale() != 1.5 {
		t.Errorf("applied %q, read as %v, drawn at %v and %v", got.GetApplied(), got.GetMode(), got.GetInterfaceScale(), got.GetTextScale())
	}
	if answer.Msg.GetInterfaceScaleBounds().GetMax() != 2 || answer.Msg.GetTextScaleBounds().GetMax() != 1.75 {
		t.Errorf("the sizes go %v and %v", answer.Msg.GetInterfaceScaleBounds(), answer.Msg.GetTextScaleBounds())
	}
	if len(answer.Msg.GetThemes()) == 0 || answer.Msg.GetThemes()[0].GetOrigin() != v1.Origin_ORIGIN_PRESET {
		t.Errorf("themes = %v", answer.Msg.GetThemes())
	}
}

func TestReadThemeAnswersWithTheStylesheetOrNothing(t *testing.T) {
	service, _, files := createService(t, themedomain.Appearance{})
	if err := writeFile(files, "kept.css", ":root { --numen-surface: #123456 }"); err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]string{"mine:kept": "#123456", "mine:gone": "", "mine:../../x": ""} {
		answer, err := service.ReadTheme(t.Context(), connect.NewRequest(&v1.ReadThemeRequest{Name: name}))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(answer.Msg.GetCss(), want) || (want == "" && answer.Msg.GetCss() != "") {
			t.Errorf("%s answered with %q", name, answer.Msg.GetCss())
		}
	}
}

func TestChoosingWritesTheThemeAndTheHalfItIsReadAs(t *testing.T) {
	service, settings, _ := createService(t, themedomain.Appearance{ThemeName: themedomain.Default})

	answer, err := service.WriteAppearance(t.Context(), connect.NewRequest(&v1.WriteAppearanceRequest{
		Name: themedomain.Default, Mode: v1.Mode_MODE_DARK,
	}))
	if err != nil || answer.Msg.GetError() != "" {
		t.Fatalf("failed: %v %q", err, answer.Msg.GetError())
	}
	want := themedomain.Appearance{ThemeName: themedomain.Default, Mode: themedomain.Dark}
	if len(settings.saves) != 1 || settings.saves[0] != want {
		t.Errorf("written: %+v", settings.saves)
	}
}

// The settings are unchanged, and the response carries the reason.
func TestAChoiceThatCannotBeAppliedIsRefusedAndReturnsWhy(t *testing.T) {
	service, settings, _ := createService(t, themedomain.Appearance{ThemeName: themedomain.Default})
	for _, selected := range []struct {
		name string
		mode v1.Mode
	}{
		{"mine:the-one-i-deleted", v1.Mode_MODE_DARK},
		{"mine:../../../.ssh/id_rsa", v1.Mode_MODE_DARK},
		{themedomain.Default, v1.Mode_MODE_UNSPECIFIED},
	} {
		answer, err := service.WriteAppearance(t.Context(), connect.NewRequest(&v1.WriteAppearanceRequest{
			Name: selected.name, Mode: selected.mode,
		}))
		if err != nil {
			t.Fatal(err)
		}
		if failed := answer.Msg.GetError(); failed == "" || !strings.Contains(failed, selected.name) {
			t.Errorf("%s, %v failed with %q", selected.name, selected.mode, failed)
		}
	}
	if len(settings.saves) != 0 {
		t.Errorf("written: %+v", settings.saves)
	}
}

func TestASettingsFileThatWouldNotTakeTheChoiceReturnsSo(t *testing.T) {
	service, settings, _ := createService(t, themedomain.Appearance{ThemeName: themedomain.Default})
	settings.failure = errNotWritten

	answer, err := service.WriteAppearance(t.Context(), connect.NewRequest(&v1.WriteAppearanceRequest{
		Name: themedomain.Default, Mode: v1.Mode_MODE_DARK,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(answer.Msg.GetError(), "read-only") {
		t.Errorf("failed with %q", answer.Msg.GetError())
	}
}

func writeFile(files themefiles.Files, name, body string) error {
	return os.WriteFile(filepath.Join(files.Dir(), name), []byte(body), 0o644)
}
