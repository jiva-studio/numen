package plugintest

import (
	"bytes"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

// Manifest is a plugin's manifest.json as the host reads it.
type Manifest = manifest.Manifest

// ReadManifest reads raw as the host reads a plugin's manifest.json. It fails the test where the host
// refuses the manifest or finds a key the schema does not name.
func ReadManifest(t *testing.T, raw []byte) Manifest {
	t.Helper()
	m, err := manifestjson.Read(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("read manifest.json: %v", err)
	}
	for _, warning := range m.Warnings {
		t.Errorf("manifest.json: %s", warning)
	}
	return m
}
