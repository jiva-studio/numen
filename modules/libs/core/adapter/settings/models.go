package settings

import (
	"encoding/json"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The shelves the models stand on.
const (
	shelfMachine    = "On this machine"
	shelfSize       = "By how large it is"
	shelfInFull     = "In full"
	shelfConfigured = "Named in the settings"
)

// EmbeddingModelAt, RecognitionModelAt and AgentModelAt are the settings that
// name a model. AgentAt names the program that answers.
var (
	EmbeddingModelAt   = []string{"indexing", "embedding", "model", "name"}
	RecognitionModelAt = []string{"indexing", "recognition", "recognise", "name"}
	AgentModelAt       = []string{"agent", "claude", "model"}
	AgentAt            = []string{"agent", "use"}
)

// Models are the models each setting that names one can be set to. Every
// adapter behind them hands the name it is given to a service or to a command
// line, and these are the models this application is built around.
//
// They are read against the settings in force, so each row says what its files
// are on this machine, and a model the settings name that is none of them
// stands as a row of its own.
func Models(held Config) []port.Model {
	models := make([]port.Model, 0, 12)
	models = append(models, embedding(held)...)
	models = append(models, recognising(held)...)
	models = append(models, answering(held)...)
	return models
}

// embedding is the model the vault is indexed by. The name is what a vector is
// kept under, and the width, the window and the pooling belong with it, so
// choosing one writes the model and the station that runs it together.
func embedding(held Config) []port.Model {
	offered := embed.Defaults()
	station := held.Indexing.Embedding.Indexing
	models := []port.Model{{
		Path:     EmbeddingModelAt,
		Name:     offered.Model.Name,
		Title:    offered.Model.Name,
		Shelf:    shelfMachine,
		Default:  true,
		Presence: embedded(station, offered.Model.Name),
		Writes: []port.Setting{
			setting([]string{"indexing", "embedding", "model"}, offered.Model),
			setting([]string{"indexing", "embedding", "indexing", "use"}, embed.UseLocal),
			setting(
				[]string{"indexing", "embedding", "indexing", "local", "name"},
				offered.Indexing.Local.Name,
			),
		},
	}}
	name := held.Indexing.Embedding.Model.Name
	if name == "" || name == offered.Model.Name {
		return models
	}
	return append(models, port.Model{
		Path:     EmbeddingModelAt,
		Name:     name,
		Title:    name,
		Shelf:    shelfConfigured,
		Presence: embedded(station, name),
		Writes: []port.Setting{
			setting([]string{"indexing", "embedding", "model"}, held.Indexing.Embedding.Model),
			setting([]string{"indexing", "embedding", "indexing", "use"}, station.Use),
			setting(
				[]string{"indexing", "embedding", "indexing", "local", "name"},
				station.Local.Name,
			),
		},
	})
}

// embedded is what the model named is on this machine, at the station the
// settings run it at. Which of the two a station is is `use`, and a station
// reaching a service fetches nothing whatever the model is called.
func embedded(station embed.Station, name string) port.Presence {
	if station.Use != embed.UseLocal {
		return port.NothingToFetch
	}
	return fetching(embed.Fetched(under(station, name)))
}

// under is how this station runs the model named: the folder, the file and the
// download the settings hold, under that name.
func under(station embed.Station, name string) embed.LocalModel {
	local := station.Local
	local.Name = name
	return local
}

// recognising is the model a scanned page is read by. It is named by where it
// is fetched from, so the row says what it is and the setting holds the address.
func recognising(held Config) []port.Model {
	offered := recognition.Defaults()
	cfg := held.Indexing.Recognition.Config
	models := []port.Model{{
		Path:     RecognitionModelAt,
		Name:     offered.Recognise.Name,
		Title:    "PP-OCRv6, small",
		Shelf:    shelfMachine,
		Default:  true,
		Presence: fetching(recognition.Fetched(cfg, offered.Recognise)),
		Writes:   []port.Setting{setting(RecognitionModelAt, offered.Recognise.Name)},
	}}
	name := cfg.Recognise.Name
	if name == "" || name == offered.Recognise.Name {
		return models
	}
	return append(models, port.Model{
		Path:     RecognitionModelAt,
		Name:     name,
		Title:    name,
		Shelf:    shelfConfigured,
		Presence: fetching(recognition.Fetched(cfg, cfg.Recognise)),
		Writes:   []port.Setting{setting(RecognitionModelAt, name)},
	})
}

// answering is the model the agent answers with. The command line takes a size
// on its own and a name in full, and an installation naming neither answers
// with whatever it is set up to answer with.
//
// The model is reached where the agent runs, and nothing of it is fetched here.
func answering(held Config) []port.Model {
	models := []port.Model{{
		Path:    AgentModelAt,
		Title:   "Whatever this machine answers with",
		Default: true,
	}}
	for _, one := range []string{"opus", "sonnet", "haiku"} {
		models = append(models, port.Model{
			Path: AgentModelAt, Name: one, Title: one, Shelf: shelfSize,
		})
	}
	for _, one := range []string{"claude-opus-5", "claude-sonnet-5", "claude-haiku-4-5"} {
		models = append(models, port.Model{
			Path: AgentModelAt, Name: one, Title: one, Shelf: shelfInFull,
		})
	}
	if name := held.Agent.Claude.Model; name != "" && !among(models, name) {
		models = append(models, port.Model{
			Path: AgentModelAt, Name: name, Title: name, Shelf: shelfConfigured,
		})
	}
	for at := range models {
		models[at].Writes = []port.Setting{setting(AgentModelAt, models[at].Name)}
	}
	return models
}

// among says whether one of these models is already the name given.
func among(models []port.Model, name string) bool {
	for _, one := range models {
		if one.Name == name {
			return true
		}
	}
	return false
}

// fetching is what a model this machine runs is: its files where a fetch puts
// them, or waiting to be fetched.
func fetching(there bool) port.Presence {
	if there {
		return port.Present
	}
	return port.NotFetched
}

// Agents are the programs the agent setting can name. This build reaches the
// one it has an adapter for, and a name it cannot reach is not offered.
func Agents() []port.Model {
	return []port.Model{
		{
			Path:    AgentAt,
			Name:    agent.UseClaude,
			Title:   "Claude Code",
			Default: true,
			Writes:  []port.Setting{setting(AgentAt, agent.UseClaude)},
		},
		{
			Path:   AgentAt,
			Title:  "Nothing answers",
			Writes: []port.Setting{setting(AgentAt, "")},
		},
	}
}

// setting is one setting written down. Every value here is written in this
// file, so one that cannot be written down is this file being wrong.
func setting(at []string, value any) port.Setting {
	said, err := json.Marshal(value)
	if err != nil {
		panic("settings: " + err.Error())
	}
	return port.Setting{Path: at, JSON: string(said)}
}

// Written is settings as JSON. What is handed in is what is written out, so a
// Config read from a file carries the defaults the file leaves out.
func Written(held Config) (string, error) {
	said, err := json.Marshal(held)
	if err != nil {
		return "", err
	}
	return string(said), nil
}
