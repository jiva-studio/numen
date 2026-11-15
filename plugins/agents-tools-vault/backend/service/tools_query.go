package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/query"
)

func createQueryTools(functions host.SearchFunctions) []registration {
	return []registration{
		bind(requirePermission(createReadSpec("library_query", "Query the library", createQueryDescription(functions),
			queryScopeSentence, resultLinkSentence, truncatedSentence),
			host.PermissionIndexRead),
			(*call).createQuery, (*query.UseCase).Start),
		bind(requirePermission(createReadSpec("library_query_results", "Read the results of a query",
			"Read a query library_query started: wait for a running query, or read the next page of a finished one "+
				"with its nextPageToken. Returns the same fields as library_query.",
			queryScopeSentence),
			host.PermissionIndexRead),
			(*call).createQuery, (*query.UseCase).Results),
	}
}
