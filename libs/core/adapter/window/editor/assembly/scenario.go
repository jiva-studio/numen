package assembly

import (
	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/core/usecase/floor"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"

	machineport "github.com/jiva-studio/numen/libs/core/usecase/file/machinecontract"
	embeddingport "github.com/jiva-studio/numen/libs/core/usecase/vectors/embeddingcontract"
)

type textScenarioAssembly interface {
	OpenVaults(moves note.Move) vaults.Scenarios
	OpenFileMenu() *textversion.FileMenu
	OpenFileText() *textversion.FileText
	OpenPluginData() *textversion.PluginData
}

type fileScenarioAssembly interface {
	OpenProxy() *file.OpenProxy
	GetMediaType() *file.GetMediaType
	GetRefresh() vaults.Refresh
}

type searchScenarioAssembly interface {
	NewSearch(asking embeddingport.Embedder, handleError func(error)) search.Search
	// OpenFloors is the floor use cases of the embedder that fills the index.
	OpenFloors(embedder embeddingport.Embedder, handleError func(error)) floor.Scenarios
}

// ScenarioAssembly is what this window requires for scenarios, search, and refresh.
type ScenarioAssembly interface {
	GetClock() machineport.Clock
	OpenNotes(index note.Levels) note.Scenarios
	searchScenarioAssembly
	textScenarioAssembly
	fileScenarioAssembly
}
