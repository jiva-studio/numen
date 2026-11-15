package preview

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/ranking"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/textplace"
)

func (u *UseCase) writeResults(ctx context.Context, merged []ranking.Candidate) ([]api.PreviewResult, error) {
	titles, err := u.getTitles(ctx, merged)
	if err != nil {
		return []api.PreviewResult{}, err
	}
	results := make([]api.PreviewResult, len(merged))
	for i, candidate := range merged {
		results[i] = api.PreviewResult{
			Path: candidate.Path, Title: titles[candidate.Path], Place: newPlace(candidate),
			Text: candidate.Text, Start: candidate.Start, Length: candidate.Length, Line: candidate.MatchLine + 1,
			URI:   textplace.FormatAddress(candidate.Path, candidate.Start, candidate.Length),
			Score: getRoundedScore(candidate.Score), ChunkCount: candidate.ChunkCount,
		}
	}
	return results, nil
}

func getRoundedScore(score *float64) *float64 {
	if score == nil {
		return nil
	}
	rounded := ranking.Round(*score)
	return &rounded
}

func (u *UseCase) getTitles(ctx context.Context, merged []ranking.Candidate) (map[string]string, error) {
	titles := make(map[string]string, len(merged))
	var notePaths []string
	for _, candidate := range merged {
		base := path.Base(candidate.Path)
		titles[candidate.Path] = strings.TrimSuffix(base, path.Ext(base))
		if strings.HasSuffix(candidate.Path, domain.NoteExtension) {
			notePaths = append(notePaths, candidate.Path)
		}
	}
	if len(notePaths) == 0 {
		return titles, nil
	}
	refs, err := u.notes.GetNotes(ctx, notePaths)
	if err != nil {
		return nil, fmt.Errorf("read the titles of the notes: %w", err)
	}
	for notePath, ref := range refs {
		if ref.Title != "" {
			titles[notePath] = ref.Title
		}
	}
	return titles, nil
}

func newPlace(candidate ranking.Candidate) *api.Place {
	s := textplace.GetSelector(candidate.Selector)
	if s.IsEmpty() {
		return nil
	}
	place := &api.Place{Page: s.Page, PageCount: 0, PositionSeconds: nil, EndSeconds: nil, CFI: s.CFI, Chapter: ""}
	if s.CFI != "" {
		place.Chapter = candidate.PlaceLabel
	}
	if len(s.T) > 0 {
		place.PositionSeconds = &s.T[0]
	}
	if len(s.T) > 1 {
		place.EndSeconds = &s.T[1]
	}
	return place
}
