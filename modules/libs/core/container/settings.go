package container

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/download"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Where the section each adapter is configured by stands in the settings file.
// The settings adapter owns the file and the sections about the window; every
// other section belongs to the adapter it configures, and is put beside them
// here, where every adapter is bound.
const (
	indexingAt    = "indexing"
	embeddingAt   = "embedding"
	recognitionAt = "recognition"
	agentAt       = "agent"
	importingAt   = "importing"
)

// Settings is the file a person configures this installation in: one object
// whose fields are its sections, in the order they stand in the file. The
// settings adapter owns the file and the sections about the window; what
// configures an adapter is that adapter's own shape, and the union of them is
// made here, where every adapter is bound.
type Settings struct {
	// Version is the shape of the file. Nothing reads it yet, and it is written
	// so that the day a section changes shape there is something to tell the
	// two apart.
	Version int `json:"v"`

	// Appearance is how the window is drawn.
	Appearance settings.Appearance `json:"appearance"`

	// Indexing is how a vault is made searchable.
	Indexing Indexing `json:"indexing"`

	// Agent is which agent answers in the panel, and what it may reach.
	Agent agent.Config `json:"agent"`

	// Importing is how an address a link note points at is reached, and where
	// the tools that reach it are.
	Importing download.Config `json:"importing"`

	// Titles is how a note's title and the name of its file are held together.
	Titles settings.Titles `json:"naming"`

	// Review is what a day of review is, on this person's clock. How a deck is
	// scheduled is in the vault, in the preset it points at.
	Review settings.Review `json:"review"`

	// Said is what reading the file leaves a person something to do about: a
	// number written where a setting does not go that far. Each is one line of
	// a band, which gives a line about sixty characters, and whoever read the
	// settings puts it where the person is.
	Said []string `json:"-"`
}

// The sections the settings adapter reads and puts right as it reads the file.
func (s *Settings) GetVersion() *int                    { return &s.Version }
func (s *Settings) GetAppearance() *settings.Appearance { return &s.Appearance }
func (s *Settings) GetReview() *settings.Review         { return &s.Review }

// Say leaves a person one line about what the file holds.
func (s *Settings) Say(said string) { s.Said = append(s.Said, said) }

// Sync is whether a note's title and its filename are kept as one name.
func (s Settings) Sync() bool { return s.Titles.Sync() }

// Hangs is whether a node hangs the headings of its note under it, and Parts is
// how many of those headings stand under it at once.
func (s Settings) Hangs() bool { return s.Appearance.Hangs() }
func (s Settings) Parts() int  { return s.Appearance.PartsUnderANode }

// DayStarts is how long past midnight a day of review begins.
func (s Settings) DayStarts() time.Duration {
	starts, _ := s.Review.Starts()
	return starts
}

// Settings reads the file a person configures this installation in. An
// installation nobody has configured is written down as what it is doing.
func (c Config) Settings() (Settings, error) {
	path, err := c.settingsFile()
	if err != nil {
		return Settings{}, err
	}
	return c.getSettingsAt(path)
}

// getSettingsAt is that document read from an explicit file.
func (c Config) getSettingsAt(path string) (Settings, error) {
	held := DefaultSettings()
	if err := settings.OpenAt(path, &held); err != nil {
		return Settings{}, err
	}
	return held, nil
}

// DefaultSettings is what an installation nobody has configured does.
func DefaultSettings() Settings {
	return Settings{
		Version:    settings.DefaultVersion,
		Appearance: settings.DefaultAppearance(),
		Indexing:   DefaultIndexing(),
		Agent:      agent.Defaults(),
		Importing:  download.Defaults(),
		Titles:     settings.DefaultTitles(),
		Review:     settings.DefaultReview(),
	}
}

// WriteJSON is every setting as JSON. What is handed in is what is written out,
// so a document read from a file carries the defaults the file leaves out.
func (s Settings) WriteJSON() (string, error) {
	written, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(written), nil
}

// getSettingsOrDefaults is that document, and what an installation nobody has
// configured does where the file cannot be read.
func (c Config) getSettingsOrDefaults() Settings {
	if path, err := c.settingsFile(); err == nil {
		if held, err := c.getSettingsAt(path); err == nil {
			return held
		}
	}
	return DefaultSettings()
}

// SetSettings is this configuration carrying what a settings file says.
//
// Every section of it is carried here, in one place both entry points use. A
// section an entry point leaves behind is a part of the application that does
// nothing and says nothing, since naming no model is how a person turns one
// off.
func (c Config) SetSettings(said Settings) Config {
	c.Embedding = said.Indexing.Embedding
	c.Recognition = said.Indexing.Recognition.Config
	c.Proofreading = said.Indexing.Proofreading
	c.ScanProofreading = said.Indexing.Recognition.Proofread
	c.TranscriptProofreading = said.Indexing.Transcription.Proofread
	c.Transcription = said.Indexing.Transcription.Config
	c.Transcribes = said.Indexing.Transcribes()
	c.TranscribesUnder = said.Indexing.TranscribesUnder()
	c.Agent = said.Agent
	c.Importing = said.Importing
	return c
}

// Models reads, as the window asks, the models each setting that names one can
// be set to, and the programs the agent setting can name.
//
// Each catalogue is the adapter's own: it says what its setting takes and where
// inside its section that setting stands, and where the section itself stands
// is said here. The settings are read with them, so every row is answered
// against what is in force; a file that cannot be read is answered against the
// defaults.
func (c Config) Models() func() []port.Model {
	return func() []port.Model { return getModels(c.getSettingsOrDefaults()) }
}

// getModels is that catalogue read against the settings given.
func getModels(held Settings) []port.Model {
	models := make([]port.Model, 0, 12)
	models = append(models, setUnder(
		embed.GetModels(held.Indexing.Embedding, embed.IsFetched),
		indexingAt, embeddingAt)...)
	models = append(models, setUnder(
		recognition.GetModels(held.Indexing.Recognition.Config, recognition.IsFetched),
		indexingAt, recognitionAt)...)
	models = append(models, setUnder(agent.GetModels(held.Agent), agentAt)...)
	models = append(models, setUnder(agent.GetPrograms(), agentAt)...)
	return models
}

// setUnder puts a catalogue's places where the section they belong to stands:
// the row's own, and every setting choosing it writes.
func setUnder(models []port.Model, at ...string) []port.Model {
	for one := range models {
		models[one].Path = slices.Concat(at, models[one].Path)
		for wrote := range models[one].Writes {
			models[one].Writes[wrote].Path = slices.Concat(at, models[one].Writes[wrote].Path)
		}
	}
	return models
}
