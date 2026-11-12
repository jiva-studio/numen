package notes

import (
	"context"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/notescore"
)

// Search ranks the notes of the vault by how their title, path and text contain the words of the query.
func (u *UseCase) Search(ctx context.Context, in api.NoteSearchInput) (api.NoteSearchOutput, error) {
	query := notescore.ParseQuery(in.Query)
	if query.IsEmpty() {
		return api.NoteSearchOutput{Query: in.Query, Matches: []api.SearchMatch{}, Total: 0}, nil
	}
	limit := notescore.GetLimit(in.Limit)
	paths, err := u.findPaths(ctx)
	if err != nil {
		return api.NoteSearchOutput{}, err
	}
	refs, err := u.lookup.GetNotes(ctx, paths)
	if err != nil {
		return api.NoteSearchOutput{}, err
	}
	matches, err := u.score(ctx, query, paths, refs)
	if err != nil {
		return api.NoteSearchOutput{}, err
	}
	slices.SortFunc(matches, compareMatches)
	total := len(matches)
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return api.NoteSearchOutput{Query: in.Query, Matches: matches, Total: total}, nil
}

func (u *UseCase) findPaths(ctx context.Context) ([]string, error) {
	paths, err := u.lookup.GetPathsOfType(ctx, domain.TypeNote)
	if err == nil && len(paths) > 0 {
		return paths, nil
	}
	return u.lookup.GetPathsOfType(ctx, "")
}

func (u *UseCase) score(ctx context.Context, query notescore.Query, paths []string, refs map[string]domain.NoteRef) ([]api.SearchMatch, error) {
	var matches []api.SearchMatch
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		title := domain.Basename(path)
		if ref, ok := refs[path]; ok && ref.Title != "" {
			title = ref.Title
		}
		textScore, snippet, line := u.scoreText(ctx, query, path)
		if score := query.ScoreName(title, path) + textScore; score > 0 {
			matches = append(matches, api.SearchMatch{Path: path, Title: title, Snippet: snippet, Line: line, Score: score})
		}
	}
	return matches, nil
}

func (u *UseCase) scoreText(ctx context.Context, query notescore.Query, path string) (score int, snippet string, line int) {
	data, stat, err := u.reader.Read(ctx, path)
	if err != nil {
		return 0, "", 0
	}
	return query.ScoreText(string(data), markdown.Parse(stat, data).Body)
}

func compareMatches(a, b api.SearchMatch) int {
	if a.Score != b.Score {
		return b.Score - a.Score
	}
	return strings.Compare(a.Path, b.Path)
}
