package container

import (
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading/openai"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// AgentProofreading is what the platform is told to start: the command line, the
// model it answers with, and what it is being asked to put right.
type AgentProofreading struct {
	Command     []string
	Model       string
	Instruction string
}

// Profile is the station a reading is put right at, by the name a consumer
// names it under. A name no profile carries is an error: a person who named one
// is owed the news that it is not there.
func (c Config) Profile(name string) (proofreading.Profile, error) {
	profile, held := c.Proofreading.Profiles[name]
	if !held {
		return proofreading.Profile{}, fmt.Errorf("no proofreading profile named %q", name)
	}
	if !profile.Named() {
		return proofreading.Profile{}, fmt.Errorf(
			"proofreading profile %q names no model to use with %q", name, profile.Use)
	}
	return profile, nil
}

// Proofreader is what puts a reading right at the profile named, told what it
// is correcting.
//
// Naming no profile is naming no proofreader: nothing comes back and a reading
// is used exactly as it was read. A profile that cannot be opened — no key — is
// that same absence, carrying the reason with it.
func (c Config) Proofreader(name, instruction string) (port.Proofreader, error) {
	if name == "" {
		return nil, nil
	}
	profile, err := c.Profile(name)
	if err != nil {
		return nil, err
	}

	switch profile.Use {
	case proofreading.UseService:
		client, err := openai.New(profile, instruction)
		if err != nil {
			return nil, err
		}
		return client, nil
	case proofreading.UseAgent:
		if c.AgentProofreader == nil {
			return nil, fmt.Errorf(
				"proofreading profile %q reaches a command line this application does not start", name)
		}
		return c.AgentProofreader(AgentProofreading{
			Command:     profile.Command,
			Model:       profile.Model,
			Instruction: instruction,
		})
	}
	return nil, fmt.Errorf("proofreading profile %q is used through %q, which is neither %q nor %q",
		name, profile.Use, proofreading.UseService, proofreading.UseAgent)
}

// ProofreadQueue is where batches are left for the profile named to answer
// about later. A profile with no queue leaves nothing.
func (c Config) ProofreadQueue(name, instruction string) (port.ProofreadQueue, error) {
	if name == "" {
		return nil, nil
	}
	profile, err := c.Profile(name)
	if err != nil {
		return nil, err
	}
	if profile.Use != proofreading.UseService || profile.BatchURL == "" {
		return nil, nil
	}
	client, err := openai.New(profile, instruction)
	if err != nil {
		return nil, err
	}
	return client, nil
}
