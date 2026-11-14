package savedsearch

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/searchnote"
)

var (
	errNoField = errors.New("give query, rerank or limit")
	// ErrChanged is what a change gets when the note changed between its read and its write.
	ErrChanged = errors.New("the note changed while it was updated; try again")
)

// Update writes the change of the input into a saved search note, scans it, and runs it at once. A query
// that is not valid returns its problems in the output and changes nothing.
func (u *UseCase) Update(ctx context.Context, vault domain.VaultID, in api.UpdateInput) (api.SavedOutput, error) {
	if in.Query == nil && in.Rerank == nil && in.Limit == nil {
		if err := u.checkSearch(ctx, vault, in.Path); err != nil {
			return api.SavedOutput{}, err
		}
		return api.SavedOutput{}, errNoField
	}
	problems, err := u.write(ctx, vault, in.Path, searchnote.Change{Query: in.Query, Rerank: in.Rerank, Limit: in.Limit})
	if err != nil || len(problems) > 0 {
		return newProblemOutput(problems), err
	}
	return u.runFirst(ctx, vault, in.Path)
}

// Write writes the query, the rerank text and the limit of the request into a saved search note and scans
// it, without a run: the index change runs it. A query that is not valid returns its problems, counted
// from the first line of the query, and changes nothing.
func (u *UseCase) Write(ctx context.Context, vault domain.VaultID, in api.UpdateSearchRequest) ([]api.Problem, error) {
	return u.write(ctx, vault, in.Path, searchnote.Change{Query: &in.Query, Rerank: &in.Rerank, Limit: &in.Limit})
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

func (u *UseCase) write(ctx context.Context, vault domain.VaultID, notePath string, change searchnote.Change) ([]api.Problem, error) {
	notes := u.open(vault)
	raw, stat, err := notes.Read(ctx, notePath)
	if err != nil {
		return nil, err
	}
	if !searchnote.IsSearch(stat, raw) {
		return nil, fmt.Errorf("%s is %w", notePath, searchnote.ErrNotASearch)
	}
	if change.Limit != nil && *change.Limit < 0 {
		return nil, fmt.Errorf("limit is %d: give a number of zero or more", *change.Limit)
	}
	if change.Query != nil {
		problems, err := checkQuery(u.settings().Preview, *change.Query, 0)
		if err != nil || len(problems) > 0 {
			return problems, err
		}
	}
	updated, err := searchnote.Apply(raw, change)
	if err != nil {
		return nil, err
	}
	if _, err := notes.Write(ctx, notePath, updated, stat); err != nil {
		if errors.Is(err, host.ErrStale) {
			return nil, ErrChanged
		}
		return nil, err
	}
	return []api.Problem{}, notes.Scan(ctx, []string{notePath})
}
