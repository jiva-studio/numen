// Package hostclient connects the search relevance plugin to the host clients named in the environment.
package hostclient

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/usecase/calibration"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/usecase/relevance"
)

// OpenFromEnv builds the host clients from the process environment. newRecords keeps the floor
// records in the store folder of a vault.
func OpenFromEnv(newRecords func(relevance.Folder) relevance.Records) (calibration.Hosts, error) {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return calibration.Hosts{}, err
	}
	client := host.NewClient(endpoint)
	search := host.NewSearchClient(endpoint)
	store := host.NewStoreClient(endpoint)
	notes := func(id domain.VaultID) host.NoteLookup { return client.With(id) }
	return calibration.Hosts{
		Floors: relevance.Host{
			Search: func(id domain.VaultID) relevance.Search {
				on := search.With(id)
				return relevance.Search{Index: on, Policy: on}
			},
			Vaults: notes,
			Store:  func(id domain.VaultID) relevance.Records { return newRecords(store.With(id)) },
		},
		Queue: host.NewQueueClient(endpoint),
	}, nil
}
