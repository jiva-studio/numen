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
	LowBatchTexts      = 3
	BalancedBatchTexts = 4
	HighBatchTexts     = 6
)

// GetRun takes the threads and the niceness the host resolved, and the batch size of the profile:
// Maximum batches maximumBatchTexts texts, High at most that many, the others the sizes above.
func GetRun(limits performance.Limits, maximumBatchTexts int) Run {
	batchTexts := BalancedBatchTexts
	switch limits.Profile {
	case performance.Eco:
		batchTexts = EcoBatchTexts
	case performance.Low:
		batchTexts = LowBatchTexts
	case performance.Balanced:
		// gomutants:disable-next-line BRANCH_CASE,STATEMENT_REMOVE reason="batchTexts is initialized to BalancedBatchTexts before switch"
		batchTexts = BalancedBatchTexts
	case performance.High:
		batchTexts = min(HighBatchTexts, maximumBatchTexts)
	case performance.Maximum:
		batchTexts = maximumBatchTexts
	}
	return Run{Threads: limits.Threads, BatchTexts: batchTexts, Nice: limits.Nice}
}

// GetIndexShare is the passes the index may take: one pass stays free for questions.
func GetIndexShare(passes int) int {
	return max(passes-1, 1)
}
