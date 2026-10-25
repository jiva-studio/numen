package assembly

import (
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
)

// ScenarioAssembly is what this window asks for scenarios, search, and refresh.
type ScenarioAssembly interface {
	// GetClock is what time it is.
	GetClock() machineport.Clock

	// NewSearch is how the window answers a question about the text the vault
	// holds. handleError is where a half that could not run is said.
	NewSearch(asking embeddingport.Embedder, handleError func(error)) search.Search

	// OpenNotes and OpenVaults are this installation's scenarios, built against the cache above. index brings
	// what a write touched up to date before the write answers.
	OpenNotes(index note.Levels) note.Scenarios
	OpenVaults(moves note.Move) vaults.Scenarios

	// OpenFileMenu is the menu the text versions make of a vault's files, and nil where the
	// installation keeps no text versions.
	OpenFileMenu() *textversion.FileMenu

	// GetRefresh brings named notes up to date for a window standing on no
	// vault. A window showing one asks what that vault was opened with.
	GetRefresh() vaults.Refresh
}
