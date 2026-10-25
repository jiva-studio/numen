package format

import (
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"
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
type stencilReader struct {
	body     []byte
	declared map[string]bool
	read     int
	stencil  Stencil
}

func findFaceSides(rest []section) []section {
	underCount := 0
	for underCount < len(rest) && rest[underCount].level == 3 {
		underCount++
	}
	return rest[:underCount]
}

func findFaceSideOpenings(sides []section) (map[string]bool, []int) {
	has := map[string]bool{}
	var opening []int
	for k, side := range sides {
		if (side.name != frontHeading && side.name != backHeading) || has[side.name] {
			continue
		}
		has[side.name] = true
		opening = append(opening, k)
	}
	return has, opening
}

func (r *stencilReader) readFaceSides(face *FaceTemplate, sides []section, opening []int, end int) {
	for k, opens := range opening {
		side := sides[opens]
		to := end
		if k+1 < len(opening) {
			to = sides[opening[k+1]].head
		}
		value, valueEnd := readTrimmedRun(r.body, side.from, to)
		r.read = getTrimmedEnd(r.body, side.head, side.from)
		if value != "" {
			r.read = valueEnd
		}
		if side.name == frontHeading {
			face.Front = value
		} else {
			face.Back = value
		}
	}
}

func (r *stencilReader) validateFacePlaceholders(pos int, face FaceTemplate) {
	for _, name := range placeholders(face.Front + "\n" + face.Back) {
		if r.declared[name] {
			continue
		}
		problem := onFace(pos, FaultPlaceholder, "this face places "+name+", which the stencil does not declare")
		problem.Field = name
		r.stencil.Problems = append(r.stencil.Problems, problem)
	}
}

func (r *stencilReader) validateFace(pos int, has map[string]bool, face FaceTemplate) {
	if !has[frontHeading] || !has[backHeading] {
		missing := frontHeading
		if has[frontHeading] {
			missing = backHeading
		}
		r.stencil.Problems = append(r.stencil.Problems, onFace(pos, FaultFaceSide,
			"this face has no "+missing+", and lays out nothing"))
		return
	}
	r.validateFacePlaceholders(pos, face)
}

func (r *stencilReader) addFace(sec section, secsAfter []section) {
	pos := len(r.stencil.Faces)
	face := FaceTemplate{Name: sec.name}

	sides := findFaceSides(secsAfter)
	end := sec.to
	if len(sides) > 0 {
		end = sides[len(sides)-1].to
	}

	has, opening := findFaceSideOpenings(sides)

	leadTo := end
	if len(opening) > 0 {
		leadTo = sides[opening[0]].head
	}
	lead, leadEnd := readTrimmedRun(r.body, sec.from, leadTo)
	face.Preamble = lead
	r.read = getTrimmedEnd(r.body, sec.head, sec.from)
	if lead != "" {
		r.read = leadEnd
	}

	r.readFaceSides(&face, sides, opening, end)
	r.stencil.Faces = append(r.stencil.Faces, face)
	r.validateFace(pos, has, face)
}

// ReadStencil reads a note as a stencil: the fields it declares in its
// frontmatter, and the faces its body lays out.
//
// It never fails. A face missing a side and a placeholder nobody declared are
// problems against the stencil, and everything else in the file is read.
func ReadStencil(n domain.Note) Stencil {
	body := []byte(n.Body)
	declaredFields, problems := fields(n.Frontmatter)
	declared := make(map[string]bool, len(declaredFields))
	for _, f := range declaredFields {
		declared[f] = true
	}

	r := stencilReader{
		body:     body,
		declared: declared,
		read:     0,
		stencil: Stencil{
			Ref:      n.Fingerprint,
			Fields:   declaredFields,
			Problems: problems,
			Preamble: "",
			Faces:    nil,
			Tail:     "",
		},
	}

	secs := sections(body, 2, 3)
	firstFace := len(body)
	for _, sec := range secs {
		if sec.level == 2 {
			firstFace = sec.head
			break
		}
	}
	r.stencil.Preamble = markdown.Normalise(string(body[:firstFace]))

	for i, sec := range secs {
		if sec.level != 2 {
			continue
		}
		r.addFace(sec, secs[i+1:])
	}

	if len(r.stencil.Faces) > 0 {
		r.stencil.Tail = markdown.Normalise(string(body[r.read:]))
	}
	return r.stencil
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
	observed := map[string]bool{}
	for _, entry := range list {
		name, isText := entry.(string)
		if !isText || strings.TrimSpace(name) == "" {
			continue
		}
		switch {
		case observed[name]:
			problems = append(problems, onField(name, FaultTwoFields,
				"two fields are called "+name+", and the first stands"))
		default:
			observed[name] = true
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		problems = append(problems, OnFile(FaultNoFields,
			"this stencil declares no field, so a card cut by it has nothing to be named by"))
	}
	return out, problems
}
