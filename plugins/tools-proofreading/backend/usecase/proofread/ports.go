// Package proofread proofreads a text version: it reads the version, asks the model about its lines and reports the corrected text as a new version.
package proofread

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/numbered"
)

// Host provides the host services a proofreading uses: the store folder for corrected texts, the
// text versions it reads and reports, and the queue it runs in.
type Host struct {
	Store func(domain.VaultID) host.StoreWriter
	Texts func(domain.VaultID) Texts
	Queue host.Queue
}

// Texts is the text versions of a vault: listed, read and reported.
type Texts interface {
	host.TextVersions
	host.TextReader
}

// AskModel returns the model's reply to a prompt under an instruction.
type AskModel func(ctx context.Context, instruction, prompt string) (string, error)

// Config is how a proofreading cuts a text into batches and how far it lets a correction move a
// line.
type Config struct {
	BatchSize       int
	Overlap         int
	InFlight        int
	MaxEditDistance float64
	Language        string
	Prompt          string
}

// Default proofreading sizes.
const (
	DefaultBatchSize = 40
	DefaultOverlap   = 4
	DefaultInFlight  = 4
)

// DefaultConfig is the configuration before the host passes one.
func DefaultConfig() Config {
	return Config{
		BatchSize:       DefaultBatchSize,
		Overlap:         DefaultOverlap,
		InFlight:        DefaultInFlight,
		MaxEditDistance: numbered.MaxEditDistance,
		Language:        "",
		Prompt:          "",
	}
}

// Limits of the language and the own prompt, in runes.
const (
	MaxLanguageLen = 64
	MaxPromptLen   = 2000
)
