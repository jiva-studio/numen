package settingsagent

import (
	"context"
	"errors"
	"testing"
	"time"

	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

type listingProvider struct {
	agentport.AgentProvider
	deadline time.Time
	hasLimit bool
}

func (p *listingProvider) GetPrograms(ctx context.Context, _ map[string][]string) ([]agentdomain.Program, error) {
	p.deadline, p.hasLimit = ctx.Deadline()
	return nil, context.DeadlineExceeded
}

func TestGetProgramListsProgramsWithinADeadline(t *testing.T) {
	provider := &listingProvider{}
	programs := NewPrograms(provider, func() AgentSettings { return AgentSettings{} })

	before := testsupport.GetMachine().Now()
	_, err := programs.GetProgram(t.Context(), "claude")

	if !errors.Is(err, domain.ErrUnavailable) {
		t.Errorf("error = %v, want unavailable", err)
	}
	if !provider.hasLimit || provider.deadline.After(before.Add(programsLimit+time.Second)) {
		t.Errorf("the listing had deadline %v (set: %v), want within %v", provider.deadline, provider.hasLimit, programsLimit)
	}
}

func TestGetProgramFailsForAProgramNotInstalledOrNotListed(t *testing.T) {
	provider := &staticProvider{programs: []agentdomain.Program{{Name: "claude"}}}
	programs := NewPrograms(provider, func() AgentSettings { return AgentSettings{} })

	if _, err := programs.GetProgram(t.Context(), "claude"); !errors.Is(err, domain.ErrUnavailable) {
		t.Errorf("not installed: error = %v, want unavailable", err)
	}
	if _, err := programs.GetProgram(t.Context(), "ghost"); !errors.Is(err, domain.ErrUnavailable) {
		t.Errorf("not listed: error = %v, want unavailable", err)
	}
}

type staticProvider struct {
	agentport.AgentProvider
	programs []agentdomain.Program
}

func (p *staticProvider) GetPrograms(context.Context, map[string][]string) ([]agentdomain.Program, error) {
	return p.programs, nil
}
