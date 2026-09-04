package format

import (
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// The two headings a face is made of.
const (
	frontHeading = "Front"
	backHeading  = "Back"
)

// fieldsKey is the frontmatter key a stencil declares its fields in.
const fieldsKey = "fields"

// ReadStencil reads a note as a stencil: the fields it declares in its
// frontmatter, and the faces its body lays out.
//
// It never fails. A face missing a side and a placeholder nobody declared are
// problems against the stencil, and everything else in the file is read.
func ReadStencil(n domain.Note) Stencil {
	s := Stencil{Ref: n.Fingerprint}
	s.Fields, s.Problems = fields(n.Frontmatter)

	declared := map[string]bool{}
	for _, f := range s.Fields {
		declared[f] = true
	}

	body := []byte(n.Body)
	// A face is a second-level heading and its sides are third; a first-level
	// heading on a face is text the face lays out.
	secs := sections(body, 2, 3)

	firstFace := len(body)
	for _, sec := range secs {
		if sec.level == 2 {
			firstFace = sec.head
			break
		}
	}
	s.Preamble = markdown.Normalised(string(body[:firstFace]))

	read := 0
	for i, sec := range secs {
		if sec.level != 2 {
			continue
		}

		at := len(s.Faces)
		face := FaceTemplate{Name: sec.name}

		rest := secs[i+1:]
		under := 0
		for under < len(rest) && rest[under].level == 3 {
			under++
		}
		sides := rest[:under]

		end := sec.to
		if under > 0 {
			end = sides[under-1].to
		}

		// A side is opened by the first heading of its name, and it runs to the
		// side after it. A heading of any other name stands under the side it is
		// written beneath, and so does a second heading of a name.
		has := map[string]bool{}
		var opening []int
		for k, side := range sides {
			if (side.name != frontHeading && side.name != backHeading) || has[side.name] {
				continue
			}
			has[side.name] = true
			opening = append(opening, k)
		}

		leadTo := end
		if len(opening) > 0 {
			leadTo = sides[opening[0]].head
		}
		lead, leadEnd := run(body, sec.from, leadTo)
		face.Lead = lead
		read = trimmedEnd(body, sec.head, sec.from)
		if lead != "" {
			read = leadEnd
		}

		for k, opens := range opening {
			side := sides[opens]
			to := end
			if k+1 < len(opening) {
				to = sides[opening[k+1]].head
			}
			value, valueEnd := run(body, side.from, to)
			read = trimmedEnd(body, side.head, side.from)
			if value != "" {
				read = valueEnd
			}
			if side.name == frontHeading {
				face.Front = value
			} else {
				face.Back = value
			}
		}
		s.Faces = append(s.Faces, face)

		if !has[frontHeading] || !has[backHeading] {
			missing := frontHeading
			if has[frontHeading] {
				missing = backHeading
			}
			s.Problems = append(s.Problems, onFace(at, FaultFaceSide,
				"this face has no "+missing+", and lays out nothing"))
			continue
		}

		for _, name := range placeholders(face.Front + "\n" + face.Back) {
			if declared[name] {
				continue
			}
			problem := onFace(at, FaultPlaceholder, "this face places "+name+", which the stencil does not declare")
			problem.Field = name
			s.Problems = append(s.Problems, problem)
		}
	}

	if len(s.Faces) > 0 {
		s.Tail = markdown.Normalised(string(body[read:]))
	}
	return s
}

// fields is what the frontmatter declares, in the order a person is asked for
// them. A name is compared as written, and the first of two of one name stands.
//
// An entry that is not a name at all is not a field, and every face that places
// it says so. A stencil left with no field cuts nothing, and says so.
func fields(frontmatter map[string]any) ([]string, []Problem) {
	list, _ := frontmatter[fieldsKey].([]any)

	var out []string
	var problems []Problem
	seen := map[string]bool{}
	for _, entry := range list {
		name, isText := entry.(string)
		if !isText || strings.TrimSpace(name) == "" {
			continue
		}
		switch {
		case seen[name]:
			problems = append(problems, onField(name, FaultTwoFields,
				"two fields are called "+name+", and the first stands"))
		default:
			seen[name] = true
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		problems = append(problems, OnFile(FaultNoFields,
			"this stencil declares no field, so a card cut by it has nothing to be named by"))
	}
	return out, problems
}
