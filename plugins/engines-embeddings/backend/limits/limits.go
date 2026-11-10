// Package limits holds what a performance profile lets the local model occupy.
package limits

import "github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"

// Run is the threads, the texts per pass and the niceness a profile allows.
type Run struct {
	Threads    int
	BatchTexts int
	Nice       int
}

// The most texts in one pass for each profile: a smaller pass holds a core for less time and gives
// the interface a turn between passes.
const (
	EcoBatchTexts      = 2
	BalancedBatchTexts = 4
)

// GetRun is the limits of profile on a machine of cpus cores. The maximum profile runs batches of
// maximumBatchTexts texts.
func GetRun(profile performance.Profile, cpus, maximumBatchTexts int) Run {
	shared := performance.Resolve(profile, cpus)
	batchTexts := BalancedBatchTexts
	switch profile {
	case performance.Eco:
		batchTexts = EcoBatchTexts
	case performance.Balanced:
		// gomutants:disable-next-line BRANCH_CASE,STATEMENT_REMOVE reason="batchTexts is initialized to BalancedBatchTexts before switch"
		batchTexts = BalancedBatchTexts
	case performance.Maximum:
		batchTexts = maximumBatchTexts
	}
	return Run{Threads: shared.Threads, BatchTexts: batchTexts, Nice: shared.Nice}
}

// GetIndexShare is the passes the index may take: one pass stays free for questions.
func GetIndexShare(passes int) int {
	return max(passes-1, 1)
}
