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
	// threads is the intra-op threads of the model and the workers that tokenize.
	threads int
	// batchTexts is the most texts in one pass.
	batchTexts int
	// nice is the niceness of the process.
	nice int
}

// The most texts in one pass for each profile: a smaller pass holds a core for less time and gives
// the interface a turn between passes.
const (
	ecoBatchTexts      = 2
	balancedBatchTexts = 4
)

// getRunLimits maps the profile on a machine with cpus logical CPUs to the limits of the model.
func getRunLimits(profile performance.Profile, cpus int) runLimits {
	shared := performance.Resolve(profile, cpus)
	batchTexts := balancedBatchTexts
	switch profile {
	case performance.Eco:
		batchTexts = ecoBatchTexts
	case performance.Balanced:
		batchTexts = balancedBatchTexts
	case performance.Maximum:
		batchTexts = engine.GetDefaultBatchTexts()
	}
	return runLimits{threads: shared.Threads, batchTexts: batchTexts, nice: shared.Nice}
}

// getLimitsLocked returns the limits of the profile in force on this machine.
func (s *embeddingServer) getLimitsLocked() runLimits {
	return getRunLimits(s.profile, runtime.NumCPU())
}

// applyProfileLocked applies a profile read from a Configure call: it lowers the priority of the
// process, bounds the Go workers, and drops an open model that was opened with other limits, so the
// next batch opens it with the new ones. The model closes once the batches in flight end.
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
	if s.embedder != nil && before != limits {
		retired := s.embedder
		s.embedder = nil
		go func() {
			if err := retired.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "close the embedder: %v\n", err)
			}
		}()
	}
}

// lowerPriorityLocked sets the niceness the limits name. A process cannot return to a higher
// priority than it has, so a profile that asks for one keeps the niceness in force until the
// plugin restarts.
func (s *embeddingServer) lowerPriorityLocked(limits runLimits) {
	if err := s.lowerNice(limits.nice); err != nil {
		fmt.Fprintf(os.Stderr, "set the priority of the embeddings plugin: %v\n", err)
	}
}
