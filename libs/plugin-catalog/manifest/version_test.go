package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

func TestVersionsAreOrderedByPrecedence(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"2026.10.2", "2026.9.30", 1},
		{"v1.2.3", "1.2.3", 0},
		{"1.0.0+build1", "1.0.0+build2", 0},
		{"1.0.0-alpha", "1.0.0", -1},
		{"1.0.0-alpha", "1.0.0-alpha.1", -1},
		{"1.0.0-alpha.1", "1.0.0-alpha.beta", -1},
		{"1.0.0-beta.2", "1.0.0-beta.11", -1},
		{"1.0.0-rc.1", "1.0.0-beta.11", 1},
		{"1.10.0", "1.9.0", 1},
	}
	for _, one := range cases {
		got, err := manifest.CompareVersions(one.a, one.b)
		if err != nil {
			t.Errorf("CompareVersions(%q, %q): %v", one.a, one.b, err)
			continue
		}
		if got != one.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", one.a, one.b, got, one.want)
		}
		reverse, err := manifest.CompareVersions(one.b, one.a)
		if err != nil || reverse != -one.want {
			t.Errorf("CompareVersions(%q, %q) = %d, %v; want %d", one.b, one.a, reverse, err, -one.want)
		}
	}
}

func TestAVersionThatIsNoVersionCannotBeCompared(t *testing.T) {
	for _, text := range []string{"", "1.0", "one", "1.0.0.0", "01.0.0"} {
		if _, err := manifest.CompareVersions(text, "1.0.0"); !errors.Is(err, manifest.ErrInvalidVersion) {
			t.Errorf("CompareVersions(%q) error = %v, want an invalid version", text, err)
		}
	}
}

func readWithAppVersion(t *testing.T, bounds string) error {
	t.Helper()
	_, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","appVersion":` + bounds + `}`))
	return err
}

func TestTheBoundsOfTheAppVersionAreVersionsInOrder(t *testing.T) {
	if err := readWithAppVersion(t, `{"min":"2026.1.0","max":"2027.1.0"}`); err != nil {
		t.Errorf("a valid range was refused: %v", err)
	}
	if err := readWithAppVersion(t, `{"min":"2026.1.0"}`); err != nil {
		t.Errorf("an open maximum was refused: %v", err)
	}
	if err := readWithAppVersion(t, `{"min":"soon"}`); !errors.Is(err, manifest.ErrInvalidAppVersion) {
		t.Errorf("a minimum that is no version: error = %v", err)
	}
	if err := readWithAppVersion(t, `{"min":"2.0.0","max":"1.0.0"}`); !errors.Is(err, manifest.ErrInvertedAppVersion) {
		t.Errorf("a minimum above the maximum: error = %v", err)
	}
}

func TestAnIncompatibleManifestReportsItsReason(t *testing.T) {
	read := func(text string) manifest.Manifest {
		t.Helper()
		m, err := manifestjson.Read(strings.NewReader(text))
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	head := `{"id":"a.b","name":"A","version":"1.0.0"`
	cases := []struct {
		name   string
		m      manifest.Manifest
		env    manifest.Environment
		reason manifest.IncompatibleReason
	}{
		{"another window", read(head + `,"targets":["desktop.flashcards"]}`), manifest.Environment{Target: "desktop.editor", AppVersion: "2026.10.0"}, manifest.ReasonTarget},
		{"an older application", read(head + `,"appVersion":{"min":"2026.11.0"}}`), manifest.Environment{Target: "desktop.editor", AppVersion: "2026.10.0"}, manifest.ReasonAppTooOld},
		{"a newer application", read(head + `,"appVersion":{"max":"2026.9.9"}}`), manifest.Environment{Target: "desktop.editor", AppVersion: "2026.10.0"}, manifest.ReasonAppTooNew},
		{"a newer protocol", manifest.Manifest{Protocol: manifest.ProtocolVersion + 1}, manifest.Environment{Target: "desktop.editor", AppVersion: "2026.10.0"}, manifest.ReasonProtocol},
	}
	for _, one := range cases {
		err := one.m.CheckCompatibility(one.env)

		var incompatible *manifest.IncompatibleError
		if !errors.As(err, &incompatible) || incompatible.Reason != one.reason || incompatible.Detail == "" {
			t.Errorf("%s: error = %v, want reason %q with a detail", one.name, err, one.reason)
		}
		if !errors.Is(err, manifest.ErrIncompatible) {
			t.Errorf("%s: error is not ErrIncompatible", one.name)
		}
	}
}

func TestAPluginRunsWhereItsRequirementsAreMet(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(
		`{"id":"a.b","name":"A","version":"1.0.0","targets":["desktop.editor"],"appVersion":{"min":"2026.1.0","max":"2026.12.0"}}`))
	if err != nil {
		t.Fatal(err)
	}

	for _, env := range []manifest.Environment{
		{Target: "desktop.editor", AppVersion: "2026.1.0"},
		{Target: "desktop.editor", AppVersion: "2026.12.0"},
		{Target: "desktop.editor", AppVersion: ""},
	} {
		if err := m.CheckCompatibility(env); err != nil {
			t.Errorf("%+v: %v", env, err)
		}
	}
	if !m.Supports("desktop.editor") || m.Supports("desktop.flashcards") {
		t.Error("Supports does not follow the targets")
	}
}
