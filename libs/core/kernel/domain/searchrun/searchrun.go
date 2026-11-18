// Package searchrun holds the searches the core runs for plugins: their records, the policy that bounds
// them, which ones expire, how a page of results is addressed, and how an error of a query reads as
// problems.
package searchrun

import (
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// ID names one search.
type ID string

// Start is a search a plugin asks to run. Owner is the plugin id. A background search shows no task.
type Start struct {
	Owner        string
	Vault        domain.Vault
	Query        string
	Priority     modelcall.Priority
	IsBackground bool
}

// Step is how far a search is.
type Step int

const (
	StepQueued Step = iota
	StepCandidates
	StepReranking
	StepDone
)

// GetStepName returns queued, candidates, reranking or done, and empty text for another value.
func GetStepName(s Step) string {
	switch s {
	case StepQueued:
		return "queued"
	case StepCandidates:
		return "candidates"
	case StepReranking:
		return "reranking"
	case StepDone:
		return "done"
	}
	return ""
}

// Problem is an error of a search. Line and Column are zero when it has no position; EndLine and EndColumn
// are the last character of a problem that covers more than one, else zero.
type Problem struct {
	Line, Column       int
	EndLine, EndColumn int
	Message            string
	Suggestion         string
}

// Record is a search as the store keeps it. Results holds the partial results while IsPartial is set, the
// final results once the search stopped, and nothing when it failed.
type Record struct {
	ID              ID
	Start           Start
	IsRunning       bool
	IsPartial       bool
	Step            Step
	Results         searchquery.SearchResults
	Errors          []Problem
	IndexGeneration string
	StartedAt       time.Time
	AccessedAt      time.Time
	Holders         int
}

// IsSameSearch reports whether two starts ask for the same search: the same owner, vault, background flag
// and query text without its surrounding space.
func IsSameSearch(a, b Start) bool {
	return a.Owner == b.Owner && a.Vault.ID == b.Vault.ID && a.IsBackground == b.IsBackground &&
		strings.TrimSpace(a.Query) == strings.TrimSpace(b.Query)
}
