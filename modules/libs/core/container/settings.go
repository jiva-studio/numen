package container

import (
	"encoding/json"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/download"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/completion"
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
	Agent settings.Agent `json:"agent"`

	// Importing is how an address a link note points at is reached.
	Importing download.Config `json:"importing"`

	// Titles is how a note's title and the name of its file are held together.
	Titles settings.Titles `json:"naming"`

	// Performance is the workload and resource calibration profile.
	Performance settings.Performance `json:"performance"`

	// Plugins is the configuration for modular plugins, keyed by plugin ID.
	Plugins map[string]map[string]any `json:"plugins,omitempty"`

	// AI is the agent and the model that answer the questions of a plugin whose
	// own setting names no agent.
	AI completion.Config `json:"ai"`

	// Notices is what reading the file leaves a person something to do about: a
	// number written where a setting does not go that far. Each is one line of
	// a band, which gives a line about sixty characters, and whoever read the
	// settings puts it where the person is.
	Notices []string `json:"-"`
}

// The sections the settings adapter reads and puts right as it reads the file.
func (s *Settings) GetVersion() *int                    { return &s.Version }
func (s *Settings) GetAppearance() *settings.Appearance { return &s.Appearance }

// AddNotice leaves a person one line about what the file holds.
func (s *Settings) AddNotice(said string) { s.Notices = append(s.Notices, said) }

// Sync is whether a note's title and its filename are kept as one name.
func (s Settings) Sync() bool { return s.Titles.Sync() }

// HasNestedHeadings is whether a node hangs the headings of its note under it, and GetNestedHeadingLimit is
// how many of those headings stand under it at once.
func (s Settings) HasNestedHeadings() bool    { return s.Appearance.HasNestedHeadings() }
func (s Settings) GetNestedHeadingLimit() int { return s.Appearance.NestedHeadingLimit }

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
		Version:     settings.DefaultVersion,
		Appearance:  settings.DefaultAppearance(),
		Indexing:    DefaultIndexing(),
		Importing:   download.Defaults(),
		Titles:      settings.DefaultTitles(),
		Performance: settings.DefaultPerformance(),
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
	c.Proofreading = said.Indexing.Proofreading
	c.ScanProofreading = said.Indexing.Recognition.Proofread
	c.TranscriptProofreading = said.Indexing.Transcription.Proofread
	c.Transcription = said.Indexing.Transcription.TranscriptionConfig
	c.ShouldTranscribe = said.Indexing.CanTranscribe()
	c.TranscribesUnder = said.Indexing.GetTranscribeLimit()
	c.Agent = said.Agent
	c.Importing = said.Importing
	c.Performance = said.Performance
	c.Hardware = settings.DetectHardware()
	c.ResolvedCompute = settings.ResolveCompute(said.Performance.Profile, c.Hardware, said.Performance.Custom)
	if c.PluginHost != nil {
		if said.Plugins != nil {
			c.PluginHost.SetSettings(said.Plugins)
		}
		c.PluginHost.SetPersistHandler(func(pluginID string, key string, value any) error {
			writer := c.GetSettingsWriter()
			raw, err := json.Marshal(value)
			if err != nil {
				return err
			}
			return writer([]port.Setting{
				{
					Path: []string{"plugins", pluginID, key},
					JSON: string(raw),
				},
			})
		})
		c.PluginHost.SetKeyStore(c.GetKeyStore())
		c.setCompleter()
	}
	return c
}

// Plugins reads, as the window asks, the discovered plugins and their settings
// schemas.
func (c Config) Plugins() func() []port.PluginInfo {
	return func() []port.PluginInfo {
		if c.PluginHost == nil {
			return nil
		}
		var discoverer port.PluginDiscoverer = c.PluginHost
		return discoverer.GetPluginInfos()
	}
}

// Models reads, as the window asks, the models each setting that names one can
// be set to, and the programs the agent setting can name.
//
// The settings are read with them, so every row is answered against what is in
// force; a file that cannot be read is answered against the defaults. The
// programs and their models are the ones the agent plugins list.
func (c Config) Models() func() []port.Model {
	return func() []port.Model {
		held := c.getSettingsOrDefaults()
		return settings.AgentModels(agentAt, held.Agent, c.getAgentPrograms(held.Agent))
	}
}
