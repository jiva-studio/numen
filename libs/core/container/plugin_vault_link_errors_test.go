package container_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestALinkEditOfATargetTheNoteDoesNotLinkReportsTheTargetWithoutItsScheme(t *testing.T) {
	_, shared := openScannedPluginVault(t, getRivers(), container.PluginVaultHooks{})
	volga := domain.Address{Scheme: domain.SchemeName, Value: "Volga"}
	const want = "Deltas.md: no link to Volga is written here"

	_, removeErr := shared.RemoveLinks(t.Context(), "Deltas.md", volga, "", statFile(t, shared, "Deltas.md"))
	_, updateErr := shared.UpdateLinks(t.Context(), "Deltas.md", volga, domain.Link{Label: "flows into"}, statFile(t, shared, "Deltas.md"))

	for call, err := range map[string]error{"RemoveLinks": removeErr, "UpdateLinks": updateErr} {
		if err == nil || err.Error() != want {
			t.Errorf("%s of a target Deltas.md does not link answered %v, want %q", call, err, want)
		}
	}
}
