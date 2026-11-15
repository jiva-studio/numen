package manifestjson_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func TestManifestDeclaresARuntimeAsADependency(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},
		"dependencies":[{"id":"sample-runtime","kind":"runtime","environment":["GPU_*","LIB_PATH"]}]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(m.GetDependencyEnvironment(), []string{"GPU_*", "LIB_PATH"}) {
		t.Errorf("environment = %v, want [GPU_* LIB_PATH]", m.GetDependencyEnvironment())
	}
	got := m.GetDependencyRequirements("/p", nil)
	if len(got) != 1 || got[0].Kind != manifest.DependencyKindRuntime || got[0].PluginDir != "/p" {
		t.Errorf("requirements = %+v", got)
	}
}

func TestAPluginThatDeclaresARuntimeHasABackendToRunIt(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","dependencies":[{"id":"sample-runtime","kind":"runtime"}]}`))
	if !errors.Is(err, manifest.ErrNoBackend) {
		t.Errorf("error = %v, want ErrNoBackend", err)
	}
}

func TestARuntimeIsFetchedFromTheSourceOfThePlatform(t *testing.T) {
	sum, member := strings.Repeat("a", 64), strings.Repeat("b", 64)
	m, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},
		"dependencies":[{"id":"lib","kind":"runtime","sources":[
			{"platform":"linux/amd64","sourceUrl":"https://example.com/linux.tgz","checksumSha256":"` + sum + `","archiveMember":"lib.so","memberSha256":"` + member + `"},
			{"platform":"windows/amd64","sourceUrl":"https://example.com/windows.zip","checksumSha256":"` + sum + `","archiveMember":"lib.dll","memberSha256":"` + member + `"}],
			"search":["/opt/lib/lib.so*"]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	req := m.GetDependencyRequirements("/p", nil)[0]
	source, ok := req.GetSource("windows", "amd64")
	if !ok || source.SourceURL != "https://example.com/windows.zip" || source.ArchiveMember != "lib.dll" || source.MemberSHA256 != member {
		t.Errorf("windows source = %+v, %v", source, ok)
	}
	if _, ok := req.GetSource("darwin", "arm64"); ok {
		t.Error("a platform the sources do not name has a source")
	}
	if !slices.Equal(req.Search, []string{"/opt/lib/lib.so*"}) {
		t.Errorf("search = %v", req.Search)
	}
}

func TestAManifestRejectsAnInvalidPlatformSource(t *testing.T) {
	sum := strings.Repeat("a", 64)
	linux := `{"platform":"linux/amd64","sourceUrl":"https://example.com/y","checksumSha256":"` + sum + `"`
	for name, dependency := range map[string]string{
		"sources and sourceUrl":  `{"id":"x","kind":"runtime","sourceUrl":"https://example.com/x","checksumSha256":"` + sum + `","sources":[` + linux + `}]}`,
		"no platform":            `{"id":"x","kind":"runtime","sources":[{"sourceUrl":"https://example.com/y","checksumSha256":"` + sum + `"}]}`,
		"platform not os/arch":   `{"id":"x","kind":"runtime","sources":[{"platform":"linux","sourceUrl":"https://example.com/y","checksumSha256":"` + sum + `"}]}`,
		"platform twice":         `{"id":"x","kind":"runtime","sources":[` + linux + `},` + linux + `}]}`,
		"http source":            `{"id":"x","kind":"runtime","sources":[{"platform":"linux/amd64","sourceUrl":"http://example.com/y","checksumSha256":"` + sum + `"}]}`,
		"member without sha256":  `{"id":"x","kind":"runtime","sources":[` + linux + `,"archiveMember":"lib.so"}]}`,
		"sha256 without member":  `{"id":"x","kind":"runtime","sources":[` + linux + `,"memberSha256":"` + sum + `"}]}`,
		"variable with a space":  `{"id":"x","kind":"runtime","environment":["GPU PATH"]}`,
		"star inside a variable": `{"id":"x","kind":"runtime","environment":["GPU_*_PATH"]}`,
		"empty search pattern":   `{"id":"x","kind":"runtime","search":[""]}`,
	} {
		_, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},"dependencies":[` + dependency + `]}`))
		if !errors.Is(err, manifest.ErrInvalidDependency) {
			t.Errorf("%s: err = %v, want ErrInvalidDependency", name, err)
		}
	}
}
