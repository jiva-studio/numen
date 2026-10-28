package settings

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/core/adapter/settings/hardware"
)

// Indexing is how a vault is made searchable.
type Indexing struct {
	Embedding EmbeddingConfig `json:"embedding"`
}

// DefaultIndexing is what an installation nobody has configured makes a vault searchable with.
func DefaultIndexing() Indexing {
	return Indexing{
		Embedding: DefaultEmbedding(),
	}
}

// EmbeddingConfig is the embedding section of this installation's settings.
type EmbeddingConfig struct {
	MaxTokens int     `json:"max_tokens"`
	Floor     float64 `json:"floor"`
}

// DefaultEmbedding cuts chunks a model reading 256 tokens takes whole.
func DefaultEmbedding() EmbeddingConfig {
	return EmbeddingConfig{MaxTokens: 256}
}

// File is the file a person configures this installation in.
type File struct {
	Version     int                       `json:"v"`
	Appearance  Appearance                `json:"appearance"`
	Indexing    Indexing                  `json:"indexing"`
	Agent       Agent                     `json:"agent"`
	Titles      Titles                    `json:"naming"`
	Performance hardware.Performance      `json:"performance"`
	Plugins     map[string]map[string]any `json:"plugins,omitempty"`
	AI          AI                        `json:"ai"`
	Notices     []string                  `json:"-"`
}

// GetVersion returns a pointer to the file format version.
func (s *File) GetVersion() *int { return &s.Version }

// GetAppearance returns a pointer to the appearance section.
func (s *File) GetAppearance() *Appearance { return &s.Appearance }

// AddNotice leaves a person one line about what the file holds.
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

// DefaultFile is what an installation nobody has configured does.
func DefaultFile() File {
	return File{
		Version:     DefaultVersion,
		Appearance:  DefaultAppearance(),
		Indexing:    DefaultIndexing(),
		Titles:      DefaultTitles(),
		Performance: hardware.DefaultPerformance(),
	}
}
