package container

import (
	"context"
	"errors"
	"testing"
	"time"

	agentcase "github.com/jiva-studio/numen/libs/core/kernel/usecase/agent"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/completion"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
)

// listingProvider is a provider whose listing waits for its context and records its deadline.
type listingProvider struct {
	agentport.AgentProvider
	deadline time.Time
	hasLimit bool
}

func (p *listingProvider) GetPrograms(ctx context.Context, _ map[string][]string) ([]agent.Program, error) {
	p.deadline, p.hasLimit = ctx.Deadline()
	return nil, context.DeadlineExceeded
}

// A lookup of the program lists the plugins' programs under a limit of its own, and a listing that
// ends in failure leaves the program unavailable.
func TestTheListingOfProgramsForAQuestionHasALimit(t *testing.T) {
	provider := &listingProvider{}
	programs := completion.NewPrograms(provider, func() completion.AgentSettings { return completion.AgentSettings{} })

	before := testsupport.GetMachine().Now()
	_, err := programs.GetProgram(t.Context(), "claude")

	if !errors.Is(err, domain.ErrUnavailable) {
		t.Errorf("error = %v, want port.ErrUnavailable", err)
	}
	if !provider.hasLimit || provider.deadline.After(before.Add(agentcase.ProgramsLimit+time.Second)) {
		t.Errorf("the listing had deadline %v (set: %v), want within %v", provider.deadline, provider.hasLimit, agentcase.ProgramsLimit)
	}
}
