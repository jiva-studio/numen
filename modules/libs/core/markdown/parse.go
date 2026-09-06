// Package markdown turns the bytes of a note into the parts the index stores.
// It is pure: no filesystem, no clock, no database.
//
// The permitted additions to markdown are frontmatter and wikilinks. Both are
// read: the annotated records in the frontmatter, and the mentions in prose.
package markdown

import (
	"bytes"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
)

var headingRe = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*\s*$`)

// Parse reads one note. It never fails: a file that cannot be understood is
// still readable text, and refusing to index it would hide it from the user.
func Parse(ref domain.Fingerprint, raw []byte) domain.Note {
	n := domain.Note{Fingerprint: ref}

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

	// An identifier is what other notes point at, across vaults. Anything that
	// is not one is reported as a problem.
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
	// One key says what a note is, out of a closed list of five. A key with
	// nothing in it says nothing, and the note is a note.
	n.Type = domain.TypeNote
	if raw, present := n.Frontmatter["type"]; present && raw != nil {
		name, isText := raw.(string)
		switch {
		case !isText:
			problems = append(problems, "type is not text")
		case strings.TrimSpace(name) == "":
		case domain.KnownNoteType(domain.NoteType(name)):
			n.Type = domain.NoteType(name)
		default:
			problems = append(problems,
				"type "+name+" is not a note, a deck, a stencil, a preset or a link")
		}
	}
	// Where a link note points. The key is read on a link note and nowhere
	// else: an address written on any other note is the person's own key.
	if n.Type == domain.TypeLink {
		problems = append(problems, address(&n)...)
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

// address reads where a link note points. A link note with nowhere to point is
// a note whose whole subject is missing, so it is said rather than guessed at,
// and the note is read as every other note is.
func address(n *domain.Note) []string {
	raw, present := n.Frontmatter["url"]
	if !present || raw == nil {
		return []string{"a link carries no url"}
	}
	written, isText := raw.(string)
	if !isText {
		return []string{"url is not text"}
	}
	at, err := domain.ParseWebAddress(written)
	if err != nil {
		return []string{"url " + written + " is not a web address"}
	}
	n.Address = at
	return nil
}

// headings walks the body a line at a time, so each heading carries the byte its
// line begins at as well as the line's number. The offset is counted over the
// bytes as they are, and a carriage return is dropped from the text alone.
func headings(body []byte) []domain.Heading {
	var out []domain.Heading
	var f Fence
	for line, at := 0, 0; at <= len(body); line++ {
		end := len(body)
		if next := bytes.IndexByte(body[at:], '\n'); next >= 0 {
			end = at + next
		}
		text := strings.TrimRight(string(body[at:end]), "\r")
		if !f.Crosses(text) && !f.Inside() {
			if m := headingRe.FindStringSubmatch(text); m != nil {
				out = append(out, domain.Heading{Level: len(m[1]), Text: m[2], Line: line, Offset: at})
			}
		}
		at = end + 1
	}
	return out
}

// title prefers an explicit frontmatter title and falls back to the filename —
// which is what the user sees in a file manager, so it is never empty.
func title(n domain.Note, notePath string) string {
	if t, ok := n.Frontmatter["title"].(string); ok && strings.TrimSpace(t) != "" {
		return strings.TrimSpace(t)
	}
	return strings.TrimSuffix(path.Base(notePath), path.Ext(notePath))
}

// Fence is where a walk down the body stands: within a code fence, or outside
// one. A block opened with backticks is closed by backticks and one opened with
// tildes by tildes, so the other mark stands inside it as text.
type Fence struct{ mark byte }

// Crosses follows one line, and reports whether that line opens or closes the
// fence.
func (f *Fence) Crosses(line string) bool {
	t := strings.TrimSpace(line)
	var mark byte
	switch {
	case strings.HasPrefix(t, "```"):
		mark = '`'
	case strings.HasPrefix(t, "~~~"):
		mark = '~'
	default:
		return false
	}
	switch f.mark {
	case 0:
		f.mark = mark
	case mark:
		f.mark = 0
	default:
		return false
	}
	return true
}

// Inside reports whether the walk stands within a fence.
func (f *Fence) Inside() bool { return f.mark != 0 }
