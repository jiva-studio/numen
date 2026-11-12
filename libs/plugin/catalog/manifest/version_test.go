package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
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
		{"1.0.0-alpha.1", "1.0.0-alpha.1", 0},
		{"1.0.0-alpha.1", "1.0.0-alpha.beta", -1},
		{"1.0.0-9", "1.0.0-1a", -1},
		{"1.0.0-beta.2", "1.0.0-beta.11", -1},
		{"1.0.0-rc.1", "1.0.0-beta.11", 1},
		{"1.10.0", "1.9.0", 1},
		{"18446744073709551615.0.0", "1.0.0", 1},
		{"1.0.0-18446744073709551615", "1.0.0-9", 1},
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
	for _, text := range []string{"", "1.0", "one", "1.0.0.0", "01.0.0", "18446744073709551616.0.0"} {
		if got, err := manifest.CompareVersions(text, "1.0.0"); !errors.Is(err, manifest.ErrInvalidVersion) || got != 0 {
			t.Errorf("CompareVersions(%q, 1.0.0) = %d, %v; want 0, invalid version", text, got, err)
		}
		if got, err := manifest.CompareVersions("1.0.0", text); !errors.Is(err, manifest.ErrInvalidVersion) || got != 0 {
			t.Errorf("CompareVersions(1.0.0, %q) = %d, %v; want 0, invalid version", text, got, err)
		}
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
		{"another window", read(head + `,"targets":["desktop.flashcards"]}`), manifest.Environment{Target: "desktop.editor"}, manifest.ReasonTarget},
		{"a newer protocol", manifest.Manifest{Protocol: manifest.ProtocolVersion + 1}, manifest.Environment{Target: "desktop.editor"}, manifest.ReasonProtocol},
	}
	for _, one := range cases {
		err := one.m.CheckCompatibility(one.env)

		var incompatible *manifest.IncompatibleError
		if !errors.As(err, &incompatible) || incompatible.Reason != one.reason || incompatible.Detail == "" {
			t.Errorf("%s: error = %v, want reason %q with a detail", one.name, err, one.reason)
		}
		if !errors.Is(err, manifest.ErrIncompatible) || errors.Is(err, manifest.ErrInvalidID) {
			t.Errorf("%s: error is not ErrIncompatible", one.name)
		}
	}
}

func TestAPluginRunsWhereItsRequirementsAreMet(t *testing.T) {
	if manifest.ProtocolVersion != 1 {
		t.Fatalf("ProtocolVersion = %d, want 1", manifest.ProtocolVersion)
	}
	if got := (manifest.Manifest{}).GetProtocol(); got != 1 {
		t.Errorf("GetProtocol() = %d, want 1", got)
	}
	if got := (manifest.Manifest{Protocol: 2}).GetProtocol(); got != 2 {
		t.Errorf("GetProtocol(2) = %d, want 2", got)
	}
	if !(manifest.Manifest{}).CanTarget("desktop.editor") {
		t.Error("empty targets should support any target")
	}
	m, err := manifestjson.Read(strings.NewReader(
		`{"id":"a.b","name":"A","version":"1.0.0","targets":["desktop.editor"]}`))
	if err != nil {
		t.Fatal(err)
	}

	env := manifest.Environment{Target: "desktop.editor"}
	if err := m.CheckCompatibility(env); err != nil {
		t.Errorf("%+v: %v", env, err)
	}
	if !m.Supports("desktop.editor") || m.Supports("desktop.flashcards") {
		t.Error("Supports does not follow the targets")
	}
}
