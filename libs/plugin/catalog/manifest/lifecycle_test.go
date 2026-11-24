package manifest_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func readBackendManifest(backend string) (manifest.Manifest, error) {
	return manifestjson.Read(strings.NewReader(`{"id":"org.example.atlas","name":"Atlas","version":"1.0.0","backend":` + backend + `}`))
}

func TestABackendThatNamesNoLifecycleIsStartedOnDemand(t *testing.T) {
	for name, backend := range map[string]string{
		"absent":   `{"entrypoint":"bin/atlas"}`,
		"onDemand": `{"entrypoint":"bin/atlas","lifecycle":"onDemand"}`,
	} {
		m, err := readBackendManifest(backend)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if m.Backend.IsPersistent() {
			t.Errorf("%s: the backend is persistent", name)
		}
	}
}

func TestABackendThatNamesThePersistentLifecycleIsPersistent(t *testing.T) {
	m, err := readBackendManifest(`{"entrypoint":"bin/atlas","lifecycle":"persistent"}`)
	if err != nil {
		t.Fatal(err)
	}

	if !m.Backend.IsPersistent() || m.Backend.Lifecycle != manifest.LifecyclePersistent {
		t.Errorf("lifecycle = %q, persistent = %v", m.Backend.Lifecycle, m.Backend.IsPersistent())
	}
	if manifest.LifecyclePersistent != "persistent" || manifest.LifecycleOnDemand != "onDemand" {
		t.Errorf("lifecycle names = %q, %q", manifest.LifecycleOnDemand, manifest.LifecyclePersistent)
	}
}

func TestAnUnknownLifecycleIsRefused(t *testing.T) {
	for _, lifecycle := range []string{"always", "Persistent", "persistent ", "on-demand"} {
		_, err := readBackendManifest(`{"entrypoint":"bin/atlas","lifecycle":"` + lifecycle + `"}`)
		if !errors.Is(err, manifest.ErrInvalidBackend) || !strings.Contains(err.Error(), "unknown lifecycle") {
			t.Errorf("lifecycle %q: error = %v, want ErrInvalidBackend naming the unknown lifecycle", lifecycle, err)
		}
	}
}

func TestAPersistentBackendWithAnIdleLimitIsRefused(t *testing.T) {
	_, err := readBackendManifest(`{"entrypoint":"bin/atlas","lifecycle":"persistent","idleStopSeconds":600}`)

	if !errors.Is(err, manifest.ErrInvalidBackend) {
		t.Errorf("error = %v, want ErrInvalidBackend", err)
	}
}

func TestAnOnDemandBackendKeepsItsIdleLimit(t *testing.T) {
	if _, err := readBackendManifest(`{"entrypoint":"bin/atlas","lifecycle":"onDemand","idleStopSeconds":600}`); err != nil {
		t.Errorf("error = %v, want none", err)
	}
}

func TestALifecycleWithoutAnEntrypointIsRefused(t *testing.T) {
	_, err := readBackendManifest(`{"lifecycle":"persistent"}`)

	if !errors.Is(err, manifest.ErrInvalidBackend) {
		t.Errorf("error = %v, want ErrInvalidBackend", err)
	}
}

func TestTheIdentityPermissionIsDeclarable(t *testing.T) {
	if manifest.PermissionIdentity != "identity" {
		t.Fatalf("PermissionIdentity = %q", manifest.PermissionIdentity)
	}
	if !slices.Contains(manifest.ListPluginPermissions(), manifest.PermissionIdentity) {
		t.Error("PluginPermissions does not list identity")
	}
	m, err := manifestjson.Read(strings.NewReader(`{"id":"org.example.atlas","name":"Atlas","version":"1.0.0","permissions":["identity"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(m.Permissions, []string{"identity"}) {
		t.Errorf("permissions = %v", m.Permissions)
	}
}
