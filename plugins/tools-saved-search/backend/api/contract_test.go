package api_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
)

// fixtureDir holds the request and response fixtures the frontend tests read as well.
const fixtureDir = "../../testdata/api"

func getContracts() map[string]plugintest.MessageContract {
	return map[string]plugintest.MessageContract{
		api.MethodGetResults: {CreateRequest: func() any { return &api.GetResultsRequest{} }, CreateResponse: func() any { return &api.GetResultsResponse{} }},
		api.MethodWatchResults: {
			CreateRequest: func() any { return &api.WatchResultsRequest{} }, CreateResponse: func() any { return &api.ResultsChange{} },
		},
		api.MethodUpdateSearch: {
			CreateRequest: func() any { return &api.UpdateSearchRequest{} }, CreateResponse: func() any { return &api.UpdateSearchResponse{} },
		},
		api.MethodListSearchFunctions: {
			CreateRequest:  func() any { return &api.ListSearchFunctionsRequest{} },
			CreateResponse: func() any { return &api.ListSearchFunctionsResponse{} },
		},
	}
}

func TestMessagesMatchFixtures(t *testing.T) {
	plugintest.CheckMessageFixtures(t, fixtureDir, getContracts())
}

func TestEveryMethodHasItsFixturePair(t *testing.T) {
	plugintest.CheckFixturePairs(t, fixtureDir, getContracts())
}

func TestTheServiceAndMethodNames(t *testing.T) {
	if api.Service != "md.numen.tools.saved.search" || api.MethodGetResults != "GetResults" ||
		api.MethodWatchResults != "WatchResults" || api.MethodUpdateSearch != "UpdateSearch" {
		t.Errorf("service %q with %q, %q, %q", api.Service, api.MethodGetResults, api.MethodWatchResults, api.MethodUpdateSearch)
	}
}
