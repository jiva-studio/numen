package embed

import (
	"encoding/json"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The shelves the models stand on.
const (
	shelfMachine    = "On this machine"
	shelfConfigured = "Named in the settings"
)

// ModelAt is where the model a vault is indexed by stands inside this adapter's
// section of the settings.
var ModelAt = []string{"model", "name"}

// GetModels are the models a vault can be indexed by, read against the settings
// in force: each row says what its files are on this machine, and a model the
// settings name that is none of them stands as a row of its own.
//
// The name is what a vector is kept under, and the width, the window and the
// pooling belong with it, so choosing one writes the model and the provider
// that runs it together.
//
// Every place named here stands inside this section, and whoever composes the
// file says where the section is.
func GetModels(held Config, isFetched func(LocalModel) bool) []port.Model {
	offered := Defaults()
	provider := held.Indexing
	offeredLocal, _ := offered.Indexing.Local()
	models := []port.Model{{
		Path:     ModelAt,
		Name:     offered.Model.Name,
		Title:    offered.Model.Name,
		Shelf:    shelfMachine,
		Default:  true,
		Presence: getPresence(provider, offered.Model.Name, isFetched),
		Writes: []port.Setting{
			newSetting([]string{"model"}, offered.Model),
			newSetting([]string{"indexing", "use"}, UseLocal),
			newSetting([]string{"indexing", "local", "name"}, offeredLocal.Name),
		},
	}}
	name := held.Model.Name
	if name == "" || name == offered.Model.Name {
		return models
	}
	writes := []port.Setting{
		newSetting([]string{"model"}, held.Model),
		newSetting([]string{"indexing", "use"}, provider.Use),
	}
	// The repository is written back where it is the one in force. A provider
	// on a service is reached by what the service calls the model, and the
	// repository beside it says nothing about this row.
	if local, ok := provider.Local(); ok {
		writes = append(writes, newSetting([]string{"indexing", "local", "name"}, local.Name))
	}
	return append(models, port.Model{
		Path:     ModelAt,
		Name:     name,
		Title:    name,
		Shelf:    shelfConfigured,
		Presence: getPresence(provider, name, isFetched),
		Writes:   writes,
	})
}

// getPresence is what the model named is on this machine, at the provider the
// settings run it at. A provider reaching a service fetches nothing whatever
// the model is called.
func getPresence(provider Provider, name string, isFetched func(LocalModel) bool) port.Presence {
	local, here := provider.Local()
	if !here {
		return port.NothingToFetch
	}
	local.Name = name
	if isFetched(local) {
		return port.Present
	}
	return port.NotFetched
}

// newSetting is one setting written down.
func newSetting(at []string, value any) port.Setting {
	said, err := json.Marshal(value)
	if err != nil {
		return port.Setting{Path: at, JSON: ""}
	}
	return port.Setting{Path: at, JSON: string(said)}
}
