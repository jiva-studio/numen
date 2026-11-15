package searchquery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
)

// fileText is the text of one file a result is read from: a note's own file, or the text version another
// source was cut from.
type fileText struct {
	prose   string
	version *domain.VersionText
	isRead  bool
}

func (p *plan) readTexts(ctx context.Context, candidates []domain.SearchResult, documents []searchquery.Document) (map[string]fileText, error) {
	paths, versions := listTextPaths(candidates, documents)
	if len(paths) == 0 {
		return map[string]fileText{}, nil
	}
	reader, err := p.u.readers.Open(p.v)
	if err != nil {
		return nil, err
	}
	var texts textdomain.Texts
	if p.u.texts != nil {
		if texts, err = p.u.texts.Open(p.v); err != nil {
			return nil, err
		}
	}
	out := make(map[string]fileText, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		text, err := readText(ctx, reader.Read, texts, path, versions[path])
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		out[path] = text
	}
	return out, nil
}

func listTextPaths(candidates []domain.SearchResult, documents []searchquery.Document) ([]string, map[string]string) {
	versions := map[string]string{}
	var paths []string
	for _, c := range candidates {
		if _, found := versions[c.Path]; !found {
			paths = append(paths, c.Path)
			versions[c.Path] = c.VersionID
		}
	}
	for _, d := range documents {
		if _, found := versions[d.Path]; !found && d.SourceKind == domain.KindNote {
			paths = append(paths, d.Path)
			versions[d.Path] = ""
		}
	}
	return paths, versions
}

func readText(
	ctx context.Context, read func(ctx context.Context, path string) ([]byte, error), texts textdomain.Texts, path, version string,
) (fileText, error) {
	if version == "" {
		raw, err := read(ctx, path)
		if edit.IsNoNote(err) {
			return fileText{}, nil
		}
		if err != nil {
			return fileText{}, err
		}
		return fileText{prose: string(raw), isRead: true}, nil
	}
	if texts == nil {
		return fileText{}, nil
	}
	doc, err := texts.GetVersionText(ctx, path, domain.TextVersionID(version))
	if errors.Is(err, textdomain.ErrNoText) {
		return fileText{}, nil
	}
	if err != nil {
		return fileText{}, err
	}
	return fileText{prose: doc.Text, version: &doc, isRead: true}, nil
}

func getSpan(text string, start, length int) string {
	if start < 0 || start >= len(text) || length <= 0 {
		return ""
	}
	return text[start:min(start+length, len(text))]
}

func getExcerpt(text string, start, length int) string {
	excerpt := getSpan(text, start, length)
	if len(excerpt) <= searchquery.MaxExcerptBytes {
		return strings.Clone(excerpt)
	}
	end := searchquery.MaxExcerptBytes
	for end > 0 && !utf8.RuneStart(excerpt[end]) {
		end--
	}
	return strings.Clone(excerpt[:end])
}

func getLine(text string, offset int) int {
	offset = min(offset, len(text))
	opens := getProseStart(text)
	if offset <= opens {
		return 0
	}
	return strings.Count(text[opens:offset], "\n")
}

func getProseStart(text string) int {
	rest := strings.TrimPrefix(text, "\ufeff")
	first, after, found := strings.Cut(rest, "\n")
	if !found || strings.TrimRight(first, "\r") != "---" {
		return 0
	}
	for pos := 0; pos < len(after); {
		line, next := after[pos:], len(after)
		if end := strings.IndexByte(after[pos:], '\n'); end >= 0 {
			line, next = after[pos:pos+end], pos+end+1
		}
		if strings.TrimRight(line, "\r") == "---" {
			return len(text) - len(after) + next
		}
		pos = next
	}
	return 0
}
