// Package markdown turns the bytes of a note into the parts the index stores.
// It is pure: no filesystem, no clock, no database.
//
// The permitted additions to markdown are frontmatter, wikilinks and line
// anchors. Only frontmatter is read here — links and anchors have no
// decided semantics yet, and anything written for them now would be rewritten.
package markdown

import (
	"bufio"
	"bytes"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

var (
	headingRe = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*\s*$`)
	// A tag is a hash followed by at least one non-digit, so that "#1" in prose
	// and "#2026" in a date are not tags. Trailing punctuation is excluded so
	// that "#physics." tags "physics".
	tagRe = regexp.MustCompile(`(^|\s)#([\p{L}][\p{L}\p{N}/_-]*)`)
)

// Parse reads one note. It never fails: a file that cannot be understood is
// still readable text, and refusing to index it would hide it from the user
// .
func Parse(ref domain.FileRef, raw []byte) domain.Note {
	n := domain.Note{Ref: ref}

	body := raw
	if fm, rest, ok := splitFrontmatter(raw); ok {
		body = rest
		if len(bytes.TrimSpace(fm)) > 0 {
			var parsed map[string]any
			if err := yaml.Unmarshal(fm, &parsed); err != nil {
				n.FrontmatterErr = err.Error()
			} else {
				n.Frontmatter = parsed
			}
		}
	}

	n.Body = string(body)
	n.Headings = headings(body)
	n.Tags = tags(n.Frontmatter, body)
	n.Title = title(n, ref.Path)
	return n
}

// splitFrontmatter returns the YAML block and the body. The opening delimiter
// must be the very first line: a `---` further down is a horizontal rule, and
// treating it as frontmatter would swallow the note.
func splitFrontmatter(raw []byte) (fm, body []byte, ok bool) {
	rest, hadBOM := bytes.CutPrefix(raw, []byte("\xef\xbb\xbf"))
	_ = hadBOM

	first, after, found := bytes.Cut(rest, []byte("\n"))
	if !found || strings.TrimRight(string(first), "\r") != "---" {
		return nil, raw, false
	}
	// The closing delimiter is a line that is exactly `---`.
	scan := after
	var collected []byte
	for {
		line, remainder, more := bytes.Cut(scan, []byte("\n"))
		if strings.TrimRight(string(line), "\r") == "---" {
			return collected, remainder, true
		}
		if !more {
			// Unterminated block: not frontmatter at all.
			return nil, raw, false
		}
		collected = append(collected, line...)
		collected = append(collected, '\n')
		scan = remainder
	}
}

func headings(body []byte) []domain.Heading {
	var out []domain.Heading
	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	pos, fenced := 0, false
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if isFence(line) {
			fenced = !fenced
			pos++
			continue
		}
		if !fenced {
			if m := headingRe.FindStringSubmatch(line); m != nil {
				out = append(out, domain.Heading{Level: len(m[1]), Text: m[2], Pos: pos})
			}
		}
		pos++
	}
	return out
}

func tags(fm map[string]any, body []byte) []string {
	seen := map[string]bool{}
	var out []string
	add := func(t string) {
		t = strings.TrimSpace(t)
		t = strings.TrimPrefix(t, "#")
		if t == "" || seen[t] {
			return
		}
		seen[t] = true
		out = append(out, t)
	}

	// `tags:` in frontmatter is read as a list or as one string, because both
	// forms are what people actually write.
	switch v := fm["tags"].(type) {
	case string:
		for _, t := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' }) {
			add(t)
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				add(s)
			}
		}
	}

	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	fenced := false
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if isFence(line) {
			fenced = !fenced
			continue
		}
		if fenced || headingRe.MatchString(line) {
			continue
		}
		for _, m := range tagRe.FindAllStringSubmatch(line, -1) {
			add(m[2])
		}
	}
	return out
}

// title prefers an explicit frontmatter title, then the first level-one
// heading, and falls back to the filename — which is what the user sees in a
// file manager, so it is never empty.
func title(n domain.Note, notePath string) string {
	if t, ok := n.Frontmatter["title"].(string); ok && strings.TrimSpace(t) != "" {
		return strings.TrimSpace(t)
	}
	for _, h := range n.Headings {
		if h.Level == 1 {
			return h.Text
		}
	}
	return strings.TrimSuffix(path.Base(notePath), path.Ext(notePath))
}

func isFence(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")
}
