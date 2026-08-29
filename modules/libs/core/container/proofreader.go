package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading/openai"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Proofreader is what puts a reading right, chosen by this installation's
// configuration.
//
// A proofreader is optional: an installation that named none answers with
// nothing, and a service that cannot be opened — no key — is that same absence,
// carrying the reason with it.
func (c Config) Proofreader() (port.Proofreader, error) {
	if !c.Proofreading.Named() {
		return nil, nil
	}
	client, err := openai.New(c.Proofreading.Service)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// ProofreadQueue is where pages are left for a proofreader to answer about
// later. An installation whose proofreader has no queue leaves nothing.
func (c Config) ProofreadQueue() (port.ProofreadQueue, error) {
	if !c.Proofreading.Named() || c.Proofreading.Service.BatchURL == "" {
		return nil, nil
	}
	client, err := openai.New(c.Proofreading.Service)
	if err != nil {
		return nil, err
	}
	return client, nil
}
