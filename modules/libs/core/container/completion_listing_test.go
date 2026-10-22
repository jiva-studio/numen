package container

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"

	"github.com/jiva-studio/numen/modules/libs/core/port/agentport"
	"github.com/jiva-studio/numen/modules/libs/core/port/completionport"
)

// listingProvider is a provider whose listing waits for its context and records its deadline.
type listingProvider struct {
	agentport.AgentProvider
	deadline time.Time
	hasLimit bool
}

func (p *listingProvider) GetPrograms(ctx context.Context, _ map[string][]string) ([]agentport.AgentProgram, error) {
	p.deadline, p.hasLimit = ctx.Deadline()
	return nil, context.DeadlineExceeded
}

// A lookup of the program lists the plugins' programs under a limit of its own, and a listing that
// ends in failure leaves the program unavailable.
func TestTheListingOfProgramsForAQuestionHasALimit(t *testing.T) {
	provider := &listingProvider{}
	programs := &completionPrograms{provider: provider, getAgent: func() settings.Agent { return settings.Agent{} }}

	before := time.Now()
	_, err := programs.GetProgram(t.Context(), "claude")

	if !errors.Is(err, completionport.ErrUnavailable) {
		t.Errorf("error = %v, want port.ErrUnavailable", err)
	}
	if !provider.hasLimit || provider.deadline.After(before.Add(programsLimit+time.Second)) {
		t.Errorf("the listing had deadline %v (set: %v), want within %v", provider.deadline, provider.hasLimit, programsLimit)
	}
}
