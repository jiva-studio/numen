package searchquery

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
)

// GetSearch returns the state of a search and one page of its results, waiting a while for it to stop.
type GetSearch struct {
	kept        keptSearches
	generations IndexGenerations
}

// Read is how a plugin reads a search. A zero Wait answers at once.
type Read struct {
	Owner     string
	ID        searchrun.ID
	Wait      time.Duration
	PageSize  int
	PageToken string
}

// Page is a search with the results of one page. NextPageToken is empty on the last page and for partial
// results; TotalResults counts the results the search holds.
type Page struct {
	Record        searchrun.Record
	IsStale       bool
	NextPageToken string
	TotalResults  int
}

// Execute returns searchrun.ErrNotFound for a search that is unknown, expired, removed or of another plugin,
// and searchrun.ErrInvalidPageToken for a token not of this search.
func (u *GetSearch) Execute(ctx context.Context, r Read) (Page, error) {
	record, stopped, err := u.kept.Find(r.Owner, r.ID)
	if err != nil {
		return Page{}, err
	}
	u.kept.Touch(r.ID)
	if r.Wait > 0 && record.IsRunning {
		if record, err = u.wait(ctx, r, stopped); err != nil {
			return Page{}, err
		}
	}
	return u.cutPage(record, r)
}

func (u *GetSearch) wait(ctx context.Context, r Read, stopped <-chan struct{}) (searchrun.Record, error) {
	select {
	case <-stopped:
	case <-u.kept.Parts().CreateTimer(min(r.Wait, u.kept.Parts().Policy.MaxWait)):
	case <-ctx.Done():
		return searchrun.Record{}, ctx.Err()
	}
	record, _, isFound := u.kept.Parts().Store.FindSearch(r.ID)
	if !isFound {
		return searchrun.Record{}, searchrun.ErrNotFound
	}
	u.kept.Touch(r.ID)
	return record, nil
}

func (u *GetSearch) cutPage(record searchrun.Record, r Read) (Page, error) {
	page := Page{
		IsStale:      record.IndexGeneration != u.generations.GetGeneration(record.Start.Vault.ID),
		TotalResults: len(record.Results.Results),
	}
	size := u.kept.Parts().Policy.GetPageSize(r.PageSize)
	offset := 0
	if r.PageToken != "" && !record.IsRunning {
		var err error
		if offset, err = searchrun.ParsePageToken(r.PageToken, record.ID, page.TotalResults); err != nil {
			return Page{}, err
		}
	}
	end := min(offset+size, page.TotalResults)
	if end < page.TotalResults && !record.IsRunning {
		page.NextPageToken = searchrun.EncodePageToken(record.ID, end)
	}
	record.Results.Results = record.Results.Results[offset:end]
	page.Record = record
	return page, nil
}
