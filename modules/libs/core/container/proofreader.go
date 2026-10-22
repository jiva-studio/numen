package container

import (
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"

	"github.com/jiva-studio/numen/modules/libs/core/port/recognitionport"
)

// ProofreaderReady says whether a proofreading plugin is available and ready.
func (c Config) ProofreaderReady() bool {
	return c.hasPlugin(plugin.CapabilityProofreading)
}

// Proofreader is what puts a recognition right at the profile named.
//
// Naming no profile is naming no proofreader: nothing comes back and a recognition
// is used exactly as it was read.
func (c Config) Proofreader(name string) (recognitionport.Proofreader, error) {
	if name == "" {
		return nil, nil
	}
	host, err := c.resolvePlugin(plugin.CapabilityProofreading)
	if err != nil {
		return nil, fmt.Errorf("proofreading %q: %w", name, err)
	}
	return plugin.OpenProofreaderBridge(host)
}

// GetScanProofreading is what a document's reading is put right with: the
// profile the settings name for a scan.
func (c Config) GetScanProofreading() run.ProofreadingConfig {
	return c.getProofreading(c.ScanProofreading)
}

// GetSpeechProofreading is the same for what a model heard.
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

// CreateRecognitionProofread is what puts a document's recognition right, and
// whether the installation named a proofreader for a scan.
func (c Config) CreateRecognitionProofread() (recognition.Proofread, bool, error) {
	return recognition.CreateProofread(c.GetScanProofreading(), c.VaultReaders(), c.GetDerivedStores())
}

// CreateTranscriptProofread is what puts a transcript right, and whether the
// installation named a proofreader for speech.
func (c Config) CreateTranscriptProofread() (transcription.Proofread, bool, error) {
	return transcription.CreateProofread(c.GetSpeechProofreading(), c.VaultReaders(), c.GetDerivedStores())
}
