package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
)

// sourceAccess supplies the active plugin's capabilities to search and indexing.
type sourceAccess struct {
	searchSource searchsourceport.Source
	watcher      indexing.DescriptionWatcher
	queries      searchquery.QuerySource
	floorHint    search.FloorHint
}

func getSourceAccess(source *sourcebridge.Source) sourceAccess {
	if source == nil {
		return sourceAccess{}
	}
	return sourceAccess{searchSource: source, watcher: source, queries: source, floorHint: source.GetFloorHint}
}
