package agent

import (
	"context"
	"fmt"
	"time"

	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
)

// ProgramsLimit is how long the providers have to list the programs they run.
const ProgramsLimit = 8 * time.Second

// programsWait is how long a read waits for the providers to list their programs. Past it the read answers
// with the programs listed last, so a provider that is slow to start does not hold the window back.
const programsWait = time.Second

// ProgramCache answers with the programs listed last when a listing is slow, and calls each onLate when a slow
// listing ends with other programs.
type ProgramCache interface {
	Get(wait time.Duration, list func() []agentdomain.Program, onLate ...func()) []agentdomain.Program
}

// ListPrograms lists the programs the providers run, with the models each offers.
type ListPrograms struct {
	provider     agentport.AgentProvider
	cache        ProgramCache
	refresh      func()
	errorHandler window.ErrorHandler
}

// NewListPrograms creates the use case over the provider and the cache of the last listing. refresh runs
// when a slow listing ends with other programs; errorHandler receives the error of a listing that failed.
func NewListPrograms(
	provider agentport.AgentProvider, cache ProgramCache, refresh func(), errorHandler window.ErrorHandler,
) ListPrograms {
	return ListPrograms{provider: provider, cache: cache, refresh: refresh, errorHandler: errorHandler}
}

// Execute lists the programs, given the command each provider runs a program with.
func (u ListPrograms) Execute(commands map[string][]string) []agentdomain.Program {
	list := func() []agentdomain.Program {
		ctx, stop := context.WithTimeout(context.Background(), ProgramsLimit)
		defer stop()
		programs, err := u.provider.GetPrograms(ctx, commands)
		if err != nil && u.errorHandler != nil {
			u.errorHandler(fmt.Errorf("list agent programs: %w", err))
		}
		return programs
	}
	return u.cache.Get(programsWait, list, u.refresh)
}
