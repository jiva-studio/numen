package markdown

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// A wikilink is the ordinary link people write. Anything inside the brackets is
// the target, an alias, or a fragment; the address parser sorts that out.
var wikilinkRe = regexp.MustCompile(`\[\[([^\]\[]+)\]\]`)

// bodyLinks finds the links written in prose. They carry no role of their own,
// so they are references — the plain "see also" of a vault.
func bodyLinks(body []byte) []domain.Link {
	var out []domain.Link
	seen := map[string]bool{}

	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var f Fence
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if f.Crosses(line) || f.Inside() {
			// A link inside a code fence is an example of a link, not one.
			continue
		}
		for _, m := range wikilinkRe.FindAllStringSubmatch(line, -1) {
			target := domain.ParseAddress(m[1])
			if target.Value == "" || seen[target.String()] {
				continue
			}
			seen[target.String()] = true
			out = append(out, domain.Link{Target: target, Role: domain.RoleRef})
		}
	}
	return out
}

// NoRole closes what is said of a `links:` entry written with no role. A role
// is what a link is for, and an entry carrying none is not read at all.
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
		to, _ := fields["to"].(string)
		if strings.TrimSpace(to) == "" {
			problems = append(problems, "links: an entry has no target")
			continue
		}
		role := domain.LinkRole(str(fields["role"]))
		if role == "" {
			problems = append(problems, "links: "+to+NoRole)
			continue
		}
		if !domain.KnownRole(role) {
			// The list of roles is closed, so an unknown one has no behaviour.
			// It is reported as a problem.
			problems = append(problems, "links: "+to+" has an unknown role "+string(role))
			continue
		}
		out = append(out, domain.Link{
			Target: domain.ParseAddress(to),
			Role:   role,
			Type:   str(fields["type"]),
			Note:   str(fields["note"]),
			Label:  str(fields["label"]),
		})
	}
	return out, problems
}

func str(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
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
