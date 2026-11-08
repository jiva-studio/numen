package service

import (
	"fmt"
	"os"
	"runtime"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/performance"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
)

// runLimits is what the performance profile lets the local model occupy.
type runLimits struct {
	threads    int
	batchTexts int
	nice       int
}

// The most texts in one pass for each profile: a smaller pass holds a core for less time and gives
// the interface a turn between passes.
const (
	ecoBatchTexts      = 2
	balancedBatchTexts = 4
)

func getRunLimits(profile performance.Profile, cpus int) runLimits {
	shared := performance.Resolve(profile, cpus)
	batchTexts := balancedBatchTexts
	switch profile {
	case performance.Eco:
		batchTexts = ecoBatchTexts
	case performance.Balanced:
		// gomutants:disable-next-line BRANCH_CASE,STATEMENT_REMOVE reason="batchTexts is initialized to balancedBatchTexts before switch"
		batchTexts = balancedBatchTexts
	case performance.Maximum:
		batchTexts = engine.GetDefaultBatchTexts()
	}
	return runLimits{threads: shared.Threads, batchTexts: batchTexts, nice: shared.Nice}
}

func (s *embeddingServer) getLimitsLocked() runLimits {
	return getRunLimits(s.profile, runtime.NumCPU())
}

func (s *embeddingServer) applyProfileLocked(profile performance.Profile) {
	if s.isProfileSet && profile == s.profile {
		return
	}
	before := s.getLimitsLocked()
	s.profile, s.isProfileSet = profile, true
	limits := s.getLimitsLocked()
	s.lowerPriorityLocked(limits)
	if s.isMock {
		return
	}
	runtime.GOMAXPROCS(limits.threads)
	if before != limits {
		s.retireOpeningLocked()
	}
	if s.embedder != nil && before != limits {
		retired := s.embedder
		s.embedder = nil
		go func() {
			// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="stderr log on background close error has no side effect"
			if err := retired.Close(); err != nil {
				// gomutants:disable-next-line STATEMENT_REMOVE reason="stderr log on background close error has no side effect"
				fmt.Fprintf(os.Stderr, "close the embedder: %v\n", err)
			}
		}()
	}
}

func (s *embeddingServer) lowerPriorityLocked(limits runLimits) {
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="stderr log on priority error has no side effect"
	if err := s.lowerNice(limits.nice); err != nil {
		// gomutants:disable-next-line STATEMENT_REMOVE reason="stderr log on priority error has no side effect"
		fmt.Fprintf(os.Stderr, "set the priority of the embeddings plugin: %v\n", err)
	}
}
