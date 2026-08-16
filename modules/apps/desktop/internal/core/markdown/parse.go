// Package markdown turns the bytes of a note into the parts the index stores.
// It is pure: no filesystem, no clock, no database.
//
// The permitted additions to markdown are frontmatter, wikilinks and line
// anchors. Frontmatter and links are read: what a link is has been decided, so
// both the annotated records and the mentions in prose are parsed. Anchors are
// not, because what an anchor addresses has not.
package markdown

import (
	"bufio"
	"bytes"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/ulid"
)

var headingRe = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*\s*$`)

// Parse reads one note. It never fails: a file that cannot be understood is
// still readable text, and refusing to index it would hide it from the user.
func Parse(ref domain.FileRef, raw []byte) domain.Note {
	n := domain.Note{Ref: ref}

	body := raw
	if frontmatter, rest, ok := splitFrontmatter(raw); ok {
		body = rest
		if len(bytes.TrimSpace(frontmatter)) > 0 {
			var parsed map[string]any
			if err := yaml.Unmarshal(frontmatter, &parsed); err != nil {
				n.FrontmatterErr = err.Error()
			} else {
				n.Frontmatter = parsed
			}
		}
	}

	n.Body = string(body)
	n.Headings = headings(body)
	n.Title = title(n, ref.Path)
	annotated, problems := frontmatterLinks(n.Frontmatter)
	n.Links = mergeLinks(annotated, bodyLinks(body))

	// An identifier is what other notes point at, across vaults, so anything
	// that is not one is reported rather than quietly becoming addressable.
	if raw, present := n.Frontmatter["id"]; present {
		id, isText := raw.(string)
		switch {
		case !isText:
			problems = append(problems, "id is not text")
		case ulid.Valid(id):
			n.ID = id
		default:
			problems = append(problems, "id "+id+" is not a ULID")
		}
	}
	n.Problems = problems
	return n
}

// splitFrontmatter returns the YAML block and the body. The opening delimiter
// must be the very first line: a `---` further down is a horizontal rule, and
// treating it as frontmatter would swallow the note.
func splitFrontmatter(raw []byte) (frontmatter, body []byte, ok bool) {
	// A byte order mark is stripped for the comparison and kept in the body,
	// because the body is what gets indexed and searched, not what gets parsed.
	rest := bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))

	first, after, found := bytes.Cut(rest, []byte("\n"))
	if !found || strings.TrimRight(string(first), "\r") != "---" {
		return nil, raw, false
	}

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

// title prefers an explicit frontmatter title, then the first level-one heading,
// and falls back to the filename — which is what the user sees in a file
// manager, so it is never empty.
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
