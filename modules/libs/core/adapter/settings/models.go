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
	shelfMachine = "On this machine"
	shelfSize    = "By how large it is"
	shelfInFull  = "In full"
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
func Models() []port.Model {
	models := make([]port.Model, 0, 10)
	models = append(models, embedding()...)
	models = append(models, recognising()...)
	models = append(models, answering()...)
	return models
}

// embedding is the model the vault is indexed by. The name is what a vector is
// kept under, and the width, the window and the pooling belong with it, so
// choosing one writes the model and the station that runs it together.
func embedding() []port.Model {
	held := embed.Defaults()
	return []port.Model{{
		NamedAt:   EmbeddingModelAt,
		Name:      held.Model.Name,
		Title:     held.Model.Name,
		Shelf:     shelfMachine,
		ByDefault: true,
		Writes: []port.Setting{
			setting([]string{"indexing", "embedding", "model"}, held.Model),
			setting([]string{"indexing", "embedding", "indexing", "use"}, embed.UseLocal),
			setting(
				[]string{"indexing", "embedding", "indexing", "local", "name"},
				held.Indexing.Local.Name,
			),
		},
	}}
}

// recognising is the model a scanned page is read by. It is named by where it
// is fetched from, so the row says what it is and the setting holds the address.
func recognising() []port.Model {
	held := recognition.Defaults()
	return []port.Model{{
		NamedAt:   RecognitionModelAt,
		Name:      held.Recognise.Name,
		Title:     "PP-OCRv6, small",
		Shelf:     shelfMachine,
		ByDefault: true,
		Writes:    []port.Setting{setting(RecognitionModelAt, held.Recognise.Name)},
	}}
}

// answering is the model the agent answers with. The command line takes a size
// on its own and a name in full, and an installation naming neither answers
// with whatever it is set up to answer with.
func answering() []port.Model {
	models := []port.Model{{
		NamedAt:   AgentModelAt,
		Title:     "Whatever this machine answers with",
		ByDefault: true,
	}}
	for _, one := range []string{"opus", "sonnet", "haiku"} {
		models = append(models, port.Model{
			NamedAt: AgentModelAt, Name: one, Title: one, Shelf: shelfSize,
		})
	}
	for _, one := range []string{"claude-opus-5", "claude-sonnet-5", "claude-haiku-4-5"} {
		models = append(models, port.Model{
			NamedAt: AgentModelAt, Name: one, Title: one, Shelf: shelfInFull,
		})
	}
	for at := range models {
		models[at].Writes = []port.Setting{setting(AgentModelAt, models[at].Name)}
	}
	return models
}

// Agents are the programs the agent setting can name. This build reaches the
// one it has an adapter for, and a name it cannot reach is not offered.
func Agents() []port.Model {
	return []port.Model{
		{
			NamedAt:   AgentAt,
			Name:      agent.UseClaude,
			Title:     "Claude Code",
			ByDefault: true,
			Writes:    []port.Setting{setting(AgentAt, agent.UseClaude)},
		},
		{
			NamedAt: AgentAt,
			Title:   "Nothing answers",
			Writes:  []port.Setting{setting(AgentAt, "")},
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
	return port.Setting{At: at, Value: string(said)}
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
