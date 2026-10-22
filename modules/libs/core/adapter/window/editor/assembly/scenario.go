package assembly

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// ScenarioAssembly is what this window asks for scenarios, rendering, extraction, search, and refresh.
type ScenarioAssembly interface {
	// GetTextExtractor takes the text out of a document and GetPageRenderer
	// draws one, and GetClock is what time it is.
	GetTextExtractor() port.TextExtractor
	GetPageRenderer() port.PageRenderer
	GetClock() port.Clock

	// NewSearch is how the window answers a question about the text the vault
	// holds. handleError is where a half that could not run is said.
	NewSearch(asking port.Embedder, handleError func(error)) search.Search

	// OpenNotes and OpenVaults are this installation's scenarios, built against the cache above. index brings
	// what a write touched up to date before the write answers.
	OpenNotes(index note.Levels) note.Scenarios
	OpenVaults(moves note.Move) vaults.Scenarios

	// OpenSourceTexts reads and writes the text made from a file of the vault.
	OpenSourceTexts() source.Texts

	// GetRefresh brings named notes up to date for a window standing on no
	// vault. A window showing one asks what that vault was opened with.
	GetRefresh() vaults.Refresh
}
