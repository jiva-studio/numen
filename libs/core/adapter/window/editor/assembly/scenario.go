package assembly

import (
	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
	"github.com/jiva-studio/numen/libs/core/usecase/trigger"
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
	OpenPluginFile() *file.OpenPluginFile
	// OpenTriggers is the triggers of the enabled plugins over the files of a vault, nil where the
	// installation runs no plugins.
	OpenTriggers() *trigger.Triggers
	// WatchPluginChanges reports each time the host applied settings, which enables, disables and
	// configures plugins, or forgot the failure of a plugin. The returned function stops the reports.
	WatchPluginChanges() (<-chan struct{}, func())
	GetMediaType() *file.GetMediaType
	GetRefresh() vaults.Refresh
}

type searchScenarioAssembly interface {
	NewSearch(asking embeddingport.Embedder, handleError func(error)) search.Search
	// SetPluginSearch lends the plugins the search index and the floors, under the embedder that fills the index.
	SetPluginSearch(embedder embeddingport.Embedder, handleError func(error))
}

// ScenarioAssembly is what this window requires for scenarios, search, and refresh.
type ScenarioAssembly interface {
	GetClock() machineport.Clock
	OpenNotes(index note.IndexUpdater) note.Scenarios
	searchScenarioAssembly
	textScenarioAssembly
	fileScenarioAssembly
}
