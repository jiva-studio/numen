package ensureindexed

import "github.com/jiva-studio/numen/libs/shared/note/domain"

// ReportRead is what happens each time a vault is read into the index.
type ReportRead struct {
	runs    *Runs
	changes *Changes
}

// NewReportRead returns the use case over the runs and the changes of the index.
func NewReportRead(runs *Runs, changes *Changes) *ReportRead {
	return &ReportRead{runs: runs, changes: changes}
}

// Execute releases the failure shown for the vault and reports a change.
func (u *ReportRead) Execute(v domain.Vault) {
	u.runs.Clear(v.ID)
	u.changes.ReportChange()
}
