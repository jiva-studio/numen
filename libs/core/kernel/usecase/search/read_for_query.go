package search

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

func (u Search) readForQuery(ctx context.Context, v domain.Vault, query string, found []domain.Passage) ([]domain.Passage, error) {
	if len(found) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	reader, err := u.readers.Open(v)
	if err != nil {
		return nil, err
	}
	of := sourceTexts{files: reader}
	if u.texts != nil {
		if of.texts, err = u.texts.Open(v); err != nil {
			return nil, err
		}
	}

	groups, order := groupBySource(found)

	fill := passageFill{
		query: query, found: found,
		filled: make([]domain.Passage, len(found)), keep: make([]bool, len(found)),
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(u.getExtractorLimit())
	for _, source := range order {
		if gctx.Err() != nil {
			break
		}
		indices := groups[source]
		g.Go(func() error {
			return fillSource(gctx, of, source, indices, fill)
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	out := make([]domain.Passage, 0, len(found))
	for i, k := range fill.keep {
		if k {
			out = append(out, fill.filled[i])
		}
	}
	return out, nil
}

func (u Search) getExtractorLimit() int {
	if u.Extractors > 0 {
		return u.Extractors
	}
	return defaultExtractors
}

// passageFill is what a search fills in: the passages found, the filled copies, the
// flags of those to keep, and the query that located them.
type passageFill struct {
	query         string
	found, filled []domain.Passage
	keep          []bool
}

func fillSource(
	ctx context.Context, of sourceTexts, source string, indices []int, fill passageFill,
) error {
	found, filled, keep, query := fill.found, fill.filled, fill.keep, fill.query
	if err := ctx.Err(); err != nil {
		return err
	}
	first := found[indices[0]]
	prose, version, err := of.getText(ctx, source, first.Producer)
	if edit.IsNoNote(err) || errors.Is(err, errUnreadable) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", source, err)
	}
	opens := proseOpens(prose)
	for _, i := range indices {
		if err := ctx.Err(); err != nil {
			return err
		}
		p := found[i]
		p.Text = span(prose, p.Start, p.Length)
		p.Line = lineOf(prose, opens, p.Start+p.HitAt)
		if version != nil {
			hitAt := p.HitAt
			if hitAt == 0 {
				hitAt = findHitOffset(p.Text, query)
			}
			p.Selector = version.GetSelector(p.Start + hitAt)
		}
		filled[i] = p
		keep[i] = true
	}
	return nil
}

func findHitOffset(text, query string) int {
	lower := strings.ToLower(text)
	best := len(text) + 1
	for word := range strings.FieldsSeq(strings.ToLower(query)) {
		if pos := strings.Index(lower, word); pos >= 0 && pos < best {
			best = pos
		}
	}
	if best > len(text) {
		return 0
	}
	return best
}

func groupBySource(found []domain.Passage) (map[string][]int, []string) {
	groups := make(map[string][]int, len(found))
	order := make([]string, 0, len(found))
	for i, p := range found {
		if _, target := groups[p.Source]; !target {
			order = append(order, p.Source)
		}
		groups[p.Source] = append(groups[p.Source], i)
	}
	return groups, order
}

// sourceTexts is where a passage's text is read from: a note's own file, and
// the text version every other source was cut from.
type sourceTexts struct {
	files vaultport.VaultReader
	texts textdomain.Texts
}

func (s sourceTexts) getText(ctx context.Context, path, version string) (string, *domain.VersionText, error) {
	if version == "" {
		raw, err := s.files.Read(ctx, path)
		if err != nil {
			return "", nil, err
		}
		return string(raw), nil, nil
	}
	if s.texts == nil {
		return "", nil, errUnreadable
	}
	doc, err := s.texts.GetVersionText(ctx, path, domain.TextVersionID(version))
	if errors.Is(err, textdomain.ErrNoText) {
		return "", nil, errUnreadable
	}
	if err != nil {
		return "", nil, err
	}
	return doc.Text, &doc, nil
}

func span(raw string, start, length int) string {
	if start < 0 || start >= len(raw) || length <= 0 {
		return ""
	}
	end := min(start+length, len(raw))
	return strings.Clone(raw[start:end])
}

func lineOf(raw string, opens, start int) int {
	if start > len(raw) {
		start = len(raw)
	}
	if start <= opens {
		return 0
	}
	return strings.Count(raw[opens:start], "\n")
}

func proseOpens(raw string) int {
	rest := strings.TrimPrefix(raw, "\ufeff")
	first, after, found := strings.Cut(rest, "\n")
	if !found || strings.TrimRight(first, "\r") != "---" {
		return 0
	}
	for pos := 0; pos < len(after); {
		lineEnd := strings.IndexByte(after[pos:], '\n')
		var line string
		var next int
		if lineEnd >= 0 {
			line = after[pos : pos+lineEnd]
			next = pos + lineEnd + 1
		} else {
			line = after[pos:]
			next = len(after)
		}
		if strings.TrimRight(line, "\r") == "---" {
			return (len(raw) - len(after)) + next
		}
		pos = next
	}
	return 0
}
