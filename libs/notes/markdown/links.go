package markdown

import (
	"bytes"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// bodyLinks finds the links written in prose. They carry no role of their own,
// so they are references — the plain "see also" of a vault.
func bodyLinks(body []byte) []domain.Link {
	var out []domain.Link
	observed := map[string]bool{}

	var f Fence
	rest := body
	for len(rest) > 0 {
		var line []byte
		if idx := bytes.IndexByte(rest, '\n'); idx >= 0 {
			line = rest[:idx]
			rest = rest[idx+1:]
		} else {
			line = rest
			rest = nil
		}
		if f.IsCrossedByBytes(line) || f.IsInside() {
			// A link inside a code fence is an example of a link, not one.
			continue
		}
		if !bytes.Contains(line, []byte("[[")) {
			continue
		}
		for _, found := range WikilinksIn(strings.TrimRight(string(line), "\r")) {
			if observed[found.Target.String()] {
				continue
			}
			observed[found.Target.String()] = true
			out = append(out, domain.Link{Target: found.Target, Role: domain.RoleRef})
		}
	}
	return out
}

// NoRole is the suffix of the message for a `links:` entry written without a
// role. A role states what a link is for, and an entry without one is not read
// at all.
const NoRole = " has no role"

// frontmatterLinks reads the `links:` block, which is where a link that carries
// a role, a type or an argument is written.
//
// A malformed entry is skipped and the rest of the note is read: everything
// else in the file still shows.
func frontmatterLinks(frontmatter map[string]any) ([]domain.Link, []string) {
	raw, ok := frontmatter["links"].([]any)
	if !ok {
		return nil, nil
	}

	var out []domain.Link
	var problems []string
	for _, entry := range raw {
		fields, ok := entry.(map[string]any)
		if !ok {
			problems = append(problems, "links: entry is not a mapping")
			continue
		}
		to := readText(fields["to"])
		if strings.TrimSpace(to) == "" {
			problems = append(problems, "links: an entry has no target")
			continue
		}
		role := domain.LinkRole(str(fields["role"]))
		if role == "" {
			problems = append(problems, "links: "+to+NoRole)
			continue
		}
		if !domain.IsKnownRole(role) {
			// The list of roles is closed, so an unknown one has no behaviour.
			// It is reported as a problem.
			problems = append(problems, "links: "+to+" has an unknown role "+string(role))
			continue
		}
		marker := domain.LinkMarker(str(fields["marker"]))
		if !domain.IsWritableMarker(marker) {
			// The list of markers is closed, so an unknown one has nothing to draw.
			// It is reported as a problem.
			problems = append(problems, "links: "+to+" has an unknown marker "+string(marker))
			continue
		}
		out = append(out, domain.Link{
			Target: domain.ParseAddress(to),
			Role:   role,
			Type:   str(fields["type"]),
			Why:    str(fields["note"]),
			Label:  str(fields["label"]),
			Marker: marker,
		})
	}
	return out, problems
}

func str(v any) string {
	return strings.TrimSpace(readText(v))
}

// readText returns v when it is a string, or "" otherwise.
func readText(v any) string {
	s, isString := v.(string)
	if !isString {
		return ""
	}
	return s
}

// mergeLinks puts the two sources together. The same target written in both
// places is one link, and the annotated record wins: a role, a type and an
// argument say more than a mention in prose.
func mergeLinks(annotated, references []domain.Link) []domain.Link {
	described := make(map[string]bool, len(annotated))
	for _, l := range annotated {
		described[l.Target.String()] = true
	}
	out := annotated
	for _, l := range references {
		if !described[l.Target.String()] {
			out = append(out, l)
		}
	}
	return out
}
