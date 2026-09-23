package recognition

import (
	"encoding/json"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The shelves the models stand on.
const (
	shelfMachine    = "On this machine"
	shelfConfigured = "Named in the settings"
)

// ModelAt is where the model a scanned page is read by stands inside this
// adapter's section of the settings.
var ModelAt = []string{"recognise", "name"}

// GetModels are the models a scanned page can be read by, read against the
// settings in force. A model is named by where it is fetched from, so the row
// says what it is and the setting holds the address.
//
// Every place named here stands inside this section, and whoever composes the
// file says where the section is.
func GetModels(held Config, isFetched func(Config, RecogniserModel) bool) []port.Model {
	offered := Defaults()
	models := []port.Model{{
		Path:      ModelAt,
		Name:      offered.Recognise.Name,
		Title:     "PP-OCRv6, small",
		Shelf:     shelfMachine,
		IsDefault: true,
		Presence:  getPresence(isFetched(held, offered.Recognise)),
		Writes:    []port.Setting{newSetting(ModelAt, offered.Recognise.Name)},
	}}
	name := held.Recognise.Name
	if name == "" || name == offered.Recognise.Name {
		return models
	}
	return append(models, port.Model{
		Path:     ModelAt,
		Name:     name,
		Title:    name,
		Shelf:    shelfConfigured,
		Presence: getPresence(isFetched(held, held.Recognise)),
		Writes:   []port.Setting{newSetting(ModelAt, name)},
	})
}

// getPresence is what a model this machine runs is: its files where a fetch
// puts them, or waiting to be fetched.
func getPresence(there bool) port.Presence {
	if there {
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
