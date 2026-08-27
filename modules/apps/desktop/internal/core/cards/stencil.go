package cards

import (
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
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
	s := Stencil{Ref: n.Ref}
	s.Fields, s.Problems = fields(n.Frontmatter)

	declared := map[string]bool{}
	for _, f := range s.Fields {
		declared[f] = true
	}

	body := []byte(n.Body)
	secs := sections(body)
	for i, sec := range secs {
		if sec.level != 2 {
			continue
		}

		at := len(s.Faces)
		face := Face{Name: sec.name}

		has := map[string]bool{}
		for _, side := range secs[i+1:] {
			if side.level != 3 {
				break
			}
			switch {
			// The first heading of a name stands.
			case has[side.name]:
			case side.name == frontHeading:
				face.Front = text(body, side.from, side.to)
			case side.name == backHeading:
				face.Back = text(body, side.from, side.to)
			}
			has[side.name] = true
		}
		s.Faces = append(s.Faces, face)

		if !has[frontHeading] || !has[backHeading] {
			missing := frontHeading
			if has[frontHeading] {
				missing = backHeading
			}
			s.Problems = append(s.Problems, onFace(at, CheckFaceSide,
				"this face has no "+missing+", and lays out nothing"))
			continue
		}

		for _, name := range placeholders(face.Front + "\n" + face.Back) {
			if declared[name] {
				continue
			}
			problem := onFace(at, CheckPlaceholder, "this face places "+name+", which the stencil does not declare")
			problem.Field = name
			s.Problems = append(s.Problems, problem)
		}
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
			problems = append(problems, onField(name, CheckTwoFields,
				"two fields are called "+name+", and the first stands"))
		default:
			seen[name] = true
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		problems = append(problems, OnFile(CheckNoFields,
			"this stencil declares no field, so a card cut by it has nothing to be named by"))
	}
	return out, problems
}
