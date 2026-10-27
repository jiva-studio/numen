package service

import (
	"context"
	"io/fs"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

type NoteSearchInput struct {
	Query string `json:"query" jsonschema:"words or phrase to search for across note titles and content"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum number of matches to return (default: 20, max: 100)"`
}

type SearchMatch struct {
	Path    string `json:"path" jsonschema:"relative path to the note"`
	Source  string `json:"source" jsonschema:"alias for path"`
	Title   string `json:"title,omitempty" jsonschema:"title of the note"`
	Snippet string `json:"snippet,omitempty" jsonschema:"matching text snippet or passage"`
	Text    string `json:"text,omitempty" jsonschema:"alias for snippet"`
	Line    int    `json:"line,omitempty" jsonschema:"1-based line number of the match in note body"`
	Score   int    `json:"score" jsonschema:"relevance score"`
}

type NoteSearchOutput struct {
	Query   string        `json:"query"`
	Matches []SearchMatch `json:"matches"`
	Total   int           `json:"total"`
}

func (t *Tools) addNoteSearch(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:        "note_search",
		Title:       "Search Notes",
		Description: "Search notes across the vault by title, path, and text content.",
		Effect:      tools.ReadOnly,
		Step:        tools.Search,
		Servers:     []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).searchNotes))
}

func (t *Tools) addSearchQuery(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:        "search_query",
		Title:       "Search Query",
		Description: "Search notes across the vault by title, path, and text content.",
		Effect:      tools.ReadOnly,
		Step:        tools.Search,
		Servers:     []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).searchNotes))
}

func (t *Tools) searchNotes(ctx context.Context, in NoteSearchInput) (NoteSearchOutput, error) {
	q := strings.TrimSpace(in.Query)
	if q == "" {
		return NoteSearchOutput{Query: in.Query, Matches: []SearchMatch{}, Total: 0}, nil
	}

	limit := normalizeSearchLimit(in.Limit)
	candidatePaths, err := t.collectNotePaths(ctx)
	if err != nil {
		return NoteSearchOutput{}, err
	}

	noteRefs, _ := t.vault.GetNotes(ctx, candidatePaths)
	queryLower := strings.ToLower(q)
	words := strings.Fields(queryLower)

	matches, err := t.evaluateMatches(ctx, candidatePaths, noteRefs, queryLower, words)
	if err != nil {
		return NoteSearchOutput{}, err
	}

	sortMatches(matches)
	total := len(matches)
	if len(matches) > limit {
		matches = matches[:limit]
	}

	return NoteSearchOutput{
		Query:   in.Query,
		Matches: matches,
		Total:   total,
	}, nil
}

func normalizeSearchLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func (t *Tools) evaluateMatches(
	ctx context.Context,
	paths []string,
	refs map[string]domain.NoteRef,
	queryLower string,
	words []string,
) ([]SearchMatch, error) {
	var matches []SearchMatch
	for _, p := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		title := domain.Basename(p)
		if ref, ok := refs[p]; ok && ref.Title != "" {
			title = ref.Title
		}

		match, ok := t.scoreNoteMatch(ctx, p, title, queryLower, words)
		if ok {
			matches = append(matches, match)
		}
	}
	return matches, nil
}

func (t *Tools) scoreNoteMatch(
	ctx context.Context,
	path string,
	title string,
	queryLower string,
	words []string,
) (SearchMatch, bool) {
	score := scoreMetadata(queryLower, words, title, path)
	contentScore, snippet, bestLine := t.scoreNoteContent(ctx, path, queryLower, words)
	score += contentScore

	if score <= 0 {
		return SearchMatch{}, false
	}

	return SearchMatch{
		Path:    path,
		Source:  path,
		Title:   title,
		Snippet: snippet,
		Text:    snippet,
		Line:    bestLine,
		Score:   score,
	}, true
}

func scoreMetadata(queryLower string, words []string, title, path string) int {
	score := 0
	titleLower := strings.ToLower(title)
	pathLower := strings.ToLower(path)

	if strings.Contains(titleLower, queryLower) {
		score += 100
	}
	if strings.Contains(pathLower, queryLower) {
		score += 50
	}
	for _, w := range words {
		if len(w) > 1 {
			if strings.Contains(titleLower, w) {
				score += 25
			}
			if strings.Contains(pathLower, w) {
				score += 15
			}
		}
	}
	return score
}

func (t *Tools) scoreNoteContent(
	ctx context.Context,
	path string,
	queryLower string,
	words []string,
) (int, string, int) {
	data, _, err := t.vault.Read(ctx, path)
	if err != nil {
		return 0, "", 0
	}

	content := string(data)
	contentLower := strings.ToLower(content)
	score := 0

	if strings.Contains(contentLower, queryLower) {
		score += 40
	}
	for _, w := range words {
		if len(w) > 1 && strings.Contains(contentLower, w) {
			score += 10
		}
	}

	snippet, bestLine := findSnippet(content, queryLower, words)
	return score, snippet, bestLine
}

func findSnippet(content, queryLower string, words []string) (string, int) {
	lines := strings.Split(content, "\n")
	snippet := ""
	bestLine := 0

	for idx, line := range lines {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, queryLower) {
			return strings.TrimSpace(line), idx + 1
		}
		if snippet == "" {
			for _, w := range words {
				if len(w) > 1 && strings.Contains(lineLower, w) {
					bestLine = idx + 1
					snippet = strings.TrimSpace(line)
					break
				}
			}
		}
	}
	return snippet, bestLine
}

func sortMatches(matches []SearchMatch) {
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		return matches[i].Path < matches[j].Path
	})
}

func (t *Tools) collectNotePaths(ctx context.Context) ([]string, error) {
	paths, err := t.vault.GetPathsOfType(ctx, domain.TypeNote)
	if err == nil && len(paths) > 0 {
		return paths, nil
	}

	untyped, err := t.vault.GetPathsOfType(ctx, "")
	if err == nil && len(untyped) > 0 {
		return untyped, nil
	}

	return t.collectDiskPaths(ctx)
}

func (t *Tools) collectDiskPaths(ctx context.Context) ([]string, error) {
	v, err := t.vault.GetVault(ctx)
	if err != nil {
		return nil, err
	}
	if v.Path == "" {
		return nil, nil
	}

	var diskPaths []string
	walkErr := filepath.WalkDir(v.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d != nil && !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			rel, relErr := filepath.Rel(v.Path, path)
			if relErr == nil {
				diskPaths = append(diskPaths, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	slices.Sort(diskPaths)
	return diskPaths, nil
}
