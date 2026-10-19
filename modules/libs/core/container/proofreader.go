package container

import (
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"
)

// getProfile is the place a recognition is put right at, by the name a consumer
// names it under.
func (c Config) getProfile(name string) (Profile, error) {
	profile, held := c.Proofreading.Profiles[name]
	if !held {
		return Profile{}, fmt.Errorf("no proofreading profile named %q", name)
	}
	return c.resolveInFlight(profile), nil
}

// resolveInFlight is the profile with the number of requests one call answers
// settled: its own, then the performance profile's LLM concurrency, then the
// default.
func (c Config) resolveInFlight(profile Profile) Profile {
	profile.InFlight = profile.GetInFlight(c.ResolvedCompute.LlmConcurrency)
	return profile
}

// ProofreaderReady says whether a proofreading plugin is available and ready.
func (c Config) ProofreaderReady() bool {
	return c.hasPlugin(plugin.CapabilityProofreading)
}

// Proofreader is what puts a recognition right at the profile named.
//
// Naming no profile is naming no proofreader: nothing comes back and a recognition
// is used exactly as it was read.
func (c Config) Proofreader(name string) (port.Proofreader, error) {
	if name == "" {
		return nil, nil
	}
	if _, err := c.getProfile(name); err != nil {
		return nil, err
	}
	host, err := c.resolvePlugin(plugin.CapabilityProofreading)
	if err != nil {
		return nil, err
	}
	return plugin.OpenProofreaderBridge(host)
}

// GetScanProofreading is what a document's reading is put right with: the
// profile the settings name for a scan, and the sizes a recognition is asked about
// at.
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
	profile := c.resolveInFlight(c.Proofreading.Profiles[config.Profile])
	return run.ProofreadingConfig{
		IsNamed:         config.Profile != "",
		IsAutomatic:     config.IsAutomatic,
		By:              func() (port.Proofreader, error) { return c.Proofreader(config.Profile) },
		Batch:           profile.BatchSize,
		Overlap:         profile.Overlap,
		InFlight:        profile.InFlight,
		MaxEditDistance: c.Proofreading.MaxEditDistance,
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
