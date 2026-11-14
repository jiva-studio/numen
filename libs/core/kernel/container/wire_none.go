package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
)

func getOperationTitles(views PluginViews) connectionport.OperationTitles {
	if views.Operations == nil {
		return none.OperationTitles{}
	}
	return views.Operations
}

func getPluginInfos(views PluginViews) settingsport.PluginInfos {
	if views.Infos == nil {
		return none.PluginInfos{}
	}
	return views.Infos
}

func getPluginChecks(views PluginViews) settingsport.PluginSettings {
	if views.Checks == nil {
		return none.PluginSettings{}
	}
	return views.Checks
}

func getSearchSource(source searchsourceport.Source) searchsourceport.Source {
	if source == nil {
		return none.SearchSource{}
	}
	return source
}
