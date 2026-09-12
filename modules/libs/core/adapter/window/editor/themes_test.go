package editor

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/container"
)

// newThemeClient is a window's themes as a client reaches them: through the handler
// that answers everything else the window asks.
func newThemeClient(t *testing.T, cfg container.Config) numenv1connect.ThemeServiceClient {
	t.Helper()

	themes, err := cfg.Themes(nil)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer((&API{Themes: themes}).NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	return numenv1connect.NewThemeServiceClient(server.Client(), server.URL)
}

// newConfig is a configuration whose files are all in one folder of a test's
// own, which is what pointing the registry somewhere does.
func newConfig(t *testing.T) container.Config {
	t.Helper()
	return container.Config{RegistryPath: filepath.Join(t.TempDir(), "vaults.json")}
}

func TestTheWindowAsksTheSameHandlerAboutItsThemes(t *testing.T) {
	client := newThemeClient(t, newConfig(t))

	answer, err := client.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetApplied() != settings.DefaultTheme {
		t.Errorf("wears %q", answer.Msg.GetApplied())
	}
	if len(answer.Msg.GetThemes()) == 0 {
		t.Error("nothing to wear")
	}

	text, err := client.ReadTheme(t.Context(),
		connect.NewRequest(&v1.ReadThemeRequest{Name: answer.Msg.GetApplied()}))
	if err != nil {
		t.Fatal(err)
	}
	if text.Msg.GetCss() == "" {
		t.Error("the theme it wears has no text")
	}
}

// The choice lands in the settings file, and the keys a person typed into it
// are still there afterwards.
func TestAThemeChosenInTheWindowIsWrittenIntoTheSettings(t *testing.T) {
	cfg := newConfig(t)
	file := filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json")
	if err := os.WriteFile(file, []byte(
		`{"indexing":{"embedding":{"indexing":{"service":{"key":"sk-the-persons-own"}}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	client := newThemeClient(t, cfg)
	chosen, err := client.WriteAppearance(t.Context(),
		connect.NewRequest(&v1.WriteAppearanceRequest{Name: "preset:nord", Mode: v1.Mode_MODE_DARK}))
	if err != nil {
		t.Fatal(err)
	}
	if reason := chosen.Msg.GetError(); reason != "" {
		t.Fatalf("refused: %s", reason)
	}

	said, err := settings.At(file)
	if err != nil {
		t.Fatal(err)
	}
	if said.Appearance.Theme != "preset:nord" || said.Appearance.Mode != settings.ModeDark {
		t.Errorf("the file says %+v", said.Appearance)
	}
	// The key sits in the half the settings are not on, which nothing reads
	// through a provider. What it is still doing there is what the file says.
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "sk-the-persons-own") {
		t.Errorf("the file came back as %s", raw)
	}

	answer, err := client.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetApplied() != "preset:nord" || answer.Msg.GetMode() != v1.Mode_MODE_DARK {
		t.Errorf("wears %q, read as %v", answer.Msg.GetApplied(), answer.Msg.GetMode())
	}
}

// The two sizes land in the settings file, and each is written on its own.
func TestASizeChosenInTheWindowIsWrittenIntoTheSettings(t *testing.T) {
	cfg := newConfig(t)
	file := filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json")
	client := newThemeClient(t, cfg)

	drawn := 1.5
	chosen, err := client.WriteAppearance(t.Context(), connect.NewRequest(&v1.WriteAppearanceRequest{
		Name:           settings.DefaultTheme,
		Mode:           v1.Mode_MODE_LIGHT,
		InterfaceScale: &drawn,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if reason := chosen.Msg.GetError(); reason != "" {
		t.Fatalf("refused: %s", reason)
	}

	said, err := settings.At(file)
	if err != nil {
		t.Fatal(err)
	}
	if said.Appearance.InterfaceScale != 1.5 || said.Appearance.TextScale != 1 {
		t.Errorf("the file says %+v", said.Appearance)
	}

	answer, err := client.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetInterfaceScale() != 1.5 || answer.Msg.GetTextScale() != 1 {
		t.Errorf("drawn at %v and set at %v", answer.Msg.GetInterfaceScale(), answer.Msg.GetTextScale())
	}
}

// A window can be made hard to read only as far as the bounds go, and the file
// is left as it stands.
func TestASizeOutsideWhatItGoesToIsRefusedAndNothingIsWritten(t *testing.T) {
	cfg := newConfig(t)
	file := filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json")
	if err := os.WriteFile(file, []byte(`{"appearance":{"theme":"preset:nord"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	client := newThemeClient(t, cfg)
	set := 4.0
	chosen, err := client.WriteAppearance(t.Context(), connect.NewRequest(&v1.WriteAppearanceRequest{
		Name:      settings.DefaultTheme,
		Mode:      v1.Mode_MODE_LIGHT,
		TextScale: &set,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if reason := chosen.Msg.GetError(); !strings.Contains(reason, "appearance.text_scale") {
		t.Errorf("refused with %q", reason)
	}

	held, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(held) != `{"appearance":{"theme":"preset:nord"}}` {
		t.Errorf("the file now reads %s", held)
	}
}

// What the command line said stands over the file, and a person choosing that
// size for themselves is what it is let go of for.
func TestASizeSaidForOneLaunchStandsUntilAPersonChoosesOne(t *testing.T) {
	cfg := newConfig(t)
	cfg.InterfaceScale = 1.25
	client := newThemeClient(t, cfg)

	answer, err := client.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetInterfaceScale() != 1.25 {
		t.Errorf("drawn at %v", answer.Msg.GetInterfaceScale())
	}

	drawn := 1.5
	if _, err := client.WriteAppearance(t.Context(), connect.NewRequest(&v1.WriteAppearanceRequest{
		Name:           settings.DefaultTheme,
		Mode:           v1.Mode_MODE_LIGHT,
		InterfaceScale: &drawn,
	})); err != nil {
		t.Fatal(err)
	}

	answer, err = client.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetInterfaceScale() != 1.5 {
		t.Errorf("drawn at %v", answer.Msg.GetInterfaceScale())
	}
}

// A build put together without a catalogue answers that it has none, and the
// rest of the window is as it was.
func TestAWindowWithNoCatalogueAnswersThatItHasNone(t *testing.T) {
	server := httptest.NewServer((&API{}).NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	client := numenv1connect.NewThemeServiceClient(server.Client(), server.URL)
	if _, err := client.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{})); err == nil {
		t.Error("a build with no catalogue listed themes")
	}
}
