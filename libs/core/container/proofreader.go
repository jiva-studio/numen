package container

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/proofreaderbridge"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/transcription"

	"github.com/jiva-studio/numen/libs/core/port/recognitionport"
)

// ProofreaderReady says whether a proofreading plugin is available and ready.
func (c Config) ProofreaderReady() bool {
	return c.hasPlugin(bridge.CapabilityProofreading)
}

// Proofreader is what puts a recognition right at the profile named.
//
// Naming no profile is naming no proofreader: nothing comes back and a recognition
// is used exactly as it was read.
func (c Config) Proofreader(name string) (recognitionport.Proofreader, error) {
	if name == "" {
		return nil, nil
	}
	host, err := c.resolvePlugin(bridge.CapabilityProofreading)
	if err != nil {
		return nil, fmt.Errorf("proofreading %q: %w", name, err)
	}
	return proofreaderbridge.Open(host)
}

// GetSpeechProofreading is what a transcript is put right with: the profile the
// settings name for speech.
func (c Config) GetSpeechProofreading() run.ProofreadingConfig {
	return c.getProofreading(c.TranscriptProofreading)
}

// getProofreading is what a recognition of one kind is put right with: the profile
// the settings name for it, opened when there is something to put right.
func (c Config) getProofreading(config Proofread) run.ProofreadingConfig {
	return run.ProofreadingConfig{
		IsNamed:     config.Profile != "",
		IsAutomatic: config.IsAutomatic,
		By:          func() (recognitionport.Proofreader, error) { return c.Proofreader(config.Profile) },
	}
}

// CreateTranscriptProofread is what puts a transcript right, and whether the
// installation named a proofreader for speech.
func (c Config) CreateTranscriptProofread() (transcription.Proofread, bool, error) {
	return transcription.CreateProofread(c.GetSpeechProofreading(), c.VaultReaders(), c.GetDerivedStores())
}
