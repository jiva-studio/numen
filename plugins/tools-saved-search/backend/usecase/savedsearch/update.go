package savedsearch

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/domain/resultcache"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/searchnote"
)

// UpdateInput is the path of a saved search note and its new query; Query is nil when the call gives none.
type UpdateInput struct {
	Path  string
	Query *string
}

var (
	errNoQuery = errors.New("give query")
	// ErrChanged is what a change gets when the note changed between its read and its write.
	ErrChanged = errors.New("the note changed while it was updated; try again")
	// ErrSuperseded is what a change gets when a newer change of the note started before its query was
	// checked.
	ErrSuperseded = errors.New("a newer change of the note replaced this one")
)

// Update writes the query of the input into a saved search note, scans it, and runs it at once. A query
// the core reports problems of returns them in the output and changes nothing.
func (u *UseCase) Update(ctx context.Context, vault domain.VaultID, in UpdateInput) (SavedSearch, error) {
	if in.Query == nil {
		if err := u.checkSearch(ctx, vault, in.Path); err != nil {
			return SavedSearch{}, err
		}
		return SavedSearch{}, errNoQuery
	}
	problems, err := u.write(ctx, vault, in.Path, *in.Query)
	if err != nil || len(problems) > 0 {
		return newProblemOutput(problems), err
	}
	return u.runFirst(ctx, vault, in.Path)
}

// Write writes the query into the saved search note at notePath, scans it, and starts its run in the
// background without waiting for it: a saved search note is not indexed, so no index change runs it. A
// query the core reports problems of returns its problems, counted from the first line of
// the query, and changes nothing. A change that a newer change of the note replaced before its query was
// checked changes nothing and returns no problem.
func (u *UseCase) Write(ctx context.Context, vault domain.VaultID, notePath string, query string) ([]resultcache.Problem, error) {
	problems, err := u.write(ctx, vault, notePath, query)
	if errors.Is(err, ErrSuperseded) {
		return []resultcache.Problem{}, nil
	}
	if err != nil || len(problems) > 0 {
		return problems, err
	}
	u.runner.StartRunNow(context.WithoutCancel(ctx), vault, notePath)
	return problems, nil
}

func (u *UseCase) checkSearch(ctx context.Context, vault domain.VaultID, notePath string) error {
	raw, stat, err := u.open(vault).Read(ctx, notePath)
	if err != nil {
		return err
	}
	if !searchnote.IsSearch(stat, raw) {
		return fmt.Errorf("%s is %w", notePath, searchnote.ErrNotASearch)
	}
	return nil
}

func (u *UseCase) write(ctx context.Context, vault domain.VaultID, notePath string, query string) ([]resultcache.Problem, error) {
	notes := u.open(vault)
	raw, stat, err := notes.Read(ctx, notePath)
	if err != nil {
		return nil, err
	}
	if !searchnote.IsSearch(stat, raw) {
		return nil, fmt.Errorf("%s is %w", notePath, searchnote.ErrNotASearch)
	}
	problems, err := u.runner.CheckNoteQuery(ctx, vault, notePath, query)
	if errors.Is(err, context.Canceled) && ctx.Err() == nil {
		return nil, ErrSuperseded
	}
	if err != nil || len(problems) > 0 {
		return problems, err
	}
	updated, err := searchnote.Apply(raw, query)
	if err != nil {
		return nil, err
	}
	if _, err := notes.Write(ctx, notePath, updated, stat); err != nil {
		if errors.Is(err, host.ErrStale) {
			return nil, ErrChanged
		}
		return nil, err
	}
	return []resultcache.Problem{}, notes.Scan(ctx, []string{notePath})
}
