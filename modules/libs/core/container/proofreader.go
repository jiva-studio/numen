package container

import (
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/agentproofreader"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"
)

var (
	errProofreadingUnavailable = errors.New("proofreading plugin is unavailable or not installed")
)

// getProfile is the place a recognition is put right at, by the name a consumer
// names it under.
func (c Config) getProfile(name string) (Profile, error) {
	profile, held := c.Proofreading.Profiles[name]
	if !held {
		return Profile{}, fmt.Errorf("no proofreading profile named %q", name)
	}
	switch profile.Use {
	case UseService, UseAgent:
	default:
		return Profile{}, fmt.Errorf(
			"proofreading profile %q is used through %q, which is neither %q nor %q",
			name, profile.Use, UseService, UseAgent)
	}
	if !profile.HasModel() {
		return Profile{}, fmt.Errorf(
			"proofreading profile %q names no model to use with %q", name, profile.Use)
	}
	return c.resolveInFlight(profile), nil
}

// resolveInFlight is the profile with the number of batches it asks about at once
// settled: its own, then the performance profile's LLM concurrency, then the
// default of its kind.
func (c Config) resolveInFlight(profile Profile) Profile {
	profile.InFlight = profile.GetInFlight(c.ResolvedCompute.LlmConcurrency)
	return profile
}

// ProofreaderReady says whether a proofreading plugin is available and ready.
func (c Config) ProofreaderReady() bool {
	if c.PluginHost != nil && !c.SafeMode {
		if _, ok := c.PluginHost.FindPluginByCapability("proofreading"); ok {
			return true
		}
	}
	return false
}

// Proofreader is what puts a recognition right at the profile named, told what it
// is proofreading.
//
// Naming no profile is naming no proofreader: nothing comes back and a recognition
// is used exactly as it was read.
func (c Config) Proofreader(name, instruction string) (port.Proofreader, error) {
	if name == "" {
		return nil, nil
	}
	profile, err := c.getProfile(name)
	if err != nil {
		return nil, err
	}

	if profile.Use == UseService {
		if host, ok := c.findPluginHost("proofreading"); ok {
			return plugin.OpenProofreaderBridge(host)
		}
		return nil, errProofreadingUnavailable
	}
	return c.openAgentProofreader(name, profile, instruction)
}

// openAgentProofreader opens a profile that reaches the program a person already has installed,
// through the program the settings name for the agent, or the default one where they name none.
func (c Config) openAgentProofreader(name string, profile Profile, instruction string) (port.Proofreader, error) {
	provider := c.AgentProvider()
	if provider == nil {
		return nil, fmt.Errorf("proofreading profile %q reaches an agent program, and no agent plugin is available", name)
	}
	if !c.Agent.IsNamed() {
		return nil, fmt.Errorf("proofreading profile %q: %w", name, settings.ErrNoAgent)
	}
	return &agentproofreader.Proofreader{
		Answerer:    provider,
		Program:     c.Agent.GetUse(),
		Command:     profile.Command,
		Model:       profile.Model,
		Instruction: instruction,
		InFlight:    profile.InFlight,
	}, nil
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
		IsNamed:     config.Profile != "",
		IsAutomatic: config.IsAutomatic,
		By:          func(what string) (port.Proofreader, error) { return c.Proofreader(config.Profile, what) },
		Queue: func(what string) (port.ProofreadQueue, error) {
			return c.ProofreadQueue(config.Profile, what)
		},
		Batch:           profile.BatchSize,
		Overlap:         profile.Overlap,
		InFlight:        profile.InFlight,
		MaxEditDistance: c.Proofreading.Distance(),
	}
}

// ProofreadQueue is where batches are left for the profile named to answer
// about later. A profile with no queue leaves nothing.
func (c Config) ProofreadQueue(name, _ string) (port.ProofreadQueue, error) {
	if name == "" {
		return nil, nil
	}
	_, err := c.getProfile(name)
	if err != nil {
		return nil, err
	}
	return nil, nil
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
