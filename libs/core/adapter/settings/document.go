package settings

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/core/adapter/settings/hardware"
)

// Indexing is how a vault is made searchable.
type Indexing struct {
	// Embedding is how the index cuts notes into chunks and stores their vectors.
	Embedding EmbeddingConfig `json:"embedding"`
}

// DefaultIndexing is the indexing configuration of an unconfigured installation.
func DefaultIndexing() Indexing {
	return Indexing{
		Embedding: DefaultEmbedding(),
	}
}

// EmbeddingConfig is the embedding section of this installation's settings.
type EmbeddingConfig struct {
	// MaxTokens is the input limit, in tokens, of one chunk the index stores a
	// vector of.
	MaxTokens int `json:"max_tokens"`
}

// DefaultEmbedding cuts chunks a model reading 256 tokens takes whole.
func DefaultEmbedding() EmbeddingConfig {
	return EmbeddingConfig{MaxTokens: 256}
}

// File is the file in which the user configures this installation.
type File struct {
	// Version is the format version of the file. Nothing reads it yet.
	Version     int                  `json:"v"`
	Appearance  Appearance           `json:"appearance"`
	Indexing    Indexing             `json:"indexing"`
	Agent       Agent                `json:"agent"`
	Titles      Titles               `json:"naming"`
	Performance hardware.Performance `json:"performance"`
	// Plugins holds the settings of each plugin, keyed by plugin id.
	Plugins map[string]map[string]any `json:"plugins,omitempty"`
	AI      AI                        `json:"ai"`
	// Notices are the warnings that reading the file produced, shown to the
	// user. They are not written to the file.
	Notices []string `json:"-"`
}

// GetVersion returns a pointer to the file format version.
func (s *File) GetVersion() *int { return &s.Version }

// GetAppearance returns a pointer to the appearance section.
func (s *File) GetAppearance() *Appearance { return &s.Appearance }

// AddNotice records one line about the file's contents, shown to the user.
func (s *File) AddNotice(notice string) { s.Notices = append(s.Notices, notice) }

// Sync is whether a note's title and its filename are kept as one name.
func (s File) Sync() bool { return s.Titles.Sync() }

// WriteJSON is every setting as JSON.
func (s File) WriteJSON() (string, error) {
	content, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// DefaultFile is the file of an unconfigured installation.
func DefaultFile() File {
	return File{
		Version:     DefaultVersion,
		Appearance:  DefaultAppearance(),
		Indexing:    DefaultIndexing(),
		Titles:      DefaultTitles(),
		Performance: hardware.DefaultPerformance(),
	}
}
