package manifest_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestABackendThatRequiresTheIdentityReportsIt(t *testing.T) {
	m, err := readBackendManifest(`{"entrypoint":"bin/atlas","requires":["identity"]}`)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(m.Backend.Requires, []string{manifest.RequirementIdentity}) || !m.Backend.HasRequirement(manifest.RequirementIdentity) {
		t.Errorf("requires = %v", m.Backend.Requires)
	}
}

func TestABackendThatNamesNoRequirementRequiresNothing(t *testing.T) {
	m, err := readBackendManifest(`{"entrypoint":"bin/atlas"}`)
	if err != nil {
		t.Fatal(err)
	}

	if m.Backend.HasRequirement(manifest.RequirementIdentity) {
		t.Error("the backend requires the identity")
	}
}

func TestAnUnknownOrRepeatedRequirementIsRefused(t *testing.T) {
	for _, requires := range []string{`["plan"]`, `["Identity"]`, `["identity","identity"]`} {
		_, err := readBackendManifest(`{"entrypoint":"bin/atlas","requires":` + requires + `}`)
		if !errors.Is(err, manifest.ErrInvalidBackend) {
			t.Errorf("requires %s: error = %v, want ErrInvalidBackend", requires, err)
		}
	}
}
