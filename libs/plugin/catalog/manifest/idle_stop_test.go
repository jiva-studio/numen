package manifest_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func readIdleStopManifest(seconds int) (manifest.Manifest, error) {
	return manifestjson.Read(strings.NewReader(`{"id":"org.example.atlas","name":"Atlas","version":"1.0.0",` +
		`"backend":{"entrypoint":"bin/atlas","idleStopSeconds":` + strconv.Itoa(seconds) + `}}`))
}

func TestABackendKeepsTheIdleLimitItsManifestSets(t *testing.T) {
	for _, seconds := range []int{manifest.MinIdleStopSeconds, 600, manifest.MaxIdleStopSeconds} {
		m, err := readIdleStopManifest(seconds)
		if err != nil {
			t.Fatalf("idleStopSeconds %d: %v", seconds, err)
		}
		if got := m.Backend.GetIdleStopTime(); got != time.Duration(seconds)*time.Second {
			t.Errorf("idleStopSeconds %d: idle limit = %v", seconds, got)
		}
	}
}

func TestABackendWithoutAnIdleLimitUsesTheLimitOfTheHost(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(`{"id":"org.example.atlas","name":"Atlas","version":"1.0.0","backend":{"entrypoint":"bin/atlas"}}`))
	if err != nil {
		t.Fatal(err)
	}

	if got := m.Backend.GetIdleStopTime(); got != 0 {
		t.Errorf("idle limit = %v, want 0", got)
	}
}

func TestAnIdleLimitOutsideTheBoundsIsRefused(t *testing.T) {
	for _, seconds := range []int{-1, manifest.MinIdleStopSeconds - 1, manifest.MaxIdleStopSeconds + 1} {
		if _, err := readIdleStopManifest(seconds); !errors.Is(err, manifest.ErrInvalidIdleStop) {
			t.Errorf("idleStopSeconds %d: error = %v, want ErrInvalidIdleStop", seconds, err)
		}
	}
}
