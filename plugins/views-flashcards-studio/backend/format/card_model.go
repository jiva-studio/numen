package format

import (
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
)

// The two headings a template is made of.
const (
	frontHeading = "Front"
	backHeading  = "Back"
)

// fieldsKey is the frontmatter key a card model declares its fields in.
const fieldsKey = "fields"

// ReadCardModel reads a note as a card model: the fields it declares in its
// frontmatter, and the templates its body lays out.
//
// It never fails. A template missing a side and a placeholder that is not
// declared are problems of the card model, and everything else in the file is read.
type cardModelReader struct {
	body      []byte
	declared  map[string]bool
	read      int
	cardModel CardModel
}

func findTemplateSides(rest []section) []section {
	underCount := 0
	for underCount < len(rest) && rest[underCount].level == 3 {
		underCount++
	}
	return rest[:underCount]
}

func findTemplateSideOpenings(sides []section) (map[string]bool, []int) {
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

func (r *cardModelReader) readTemplateSides(template *CardTemplate, sides []section, opening []int, end int) {
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
			template.Front = value
		} else {
			template.Back = value
		}
	}
}

func (r *cardModelReader) validateTemplatePlaceholders(pos int, template CardTemplate) {
	for _, name := range placeholders(template.Front + "\n" + template.Back) {
		if r.declared[name] {
			continue
		}
		problem := onTemplate(pos, FaultPlaceholder, "this template places "+name+", which the card model does not declare")
		problem.Field = name
		r.cardModel.Problems = append(r.cardModel.Problems, problem)
	}
}

func (r *cardModelReader) validateTemplate(pos int, has map[string]bool, template CardTemplate) {
	if !has[frontHeading] || !has[backHeading] {
		missing := frontHeading
		if has[frontHeading] {
			missing = backHeading
		}
		r.cardModel.Problems = append(r.cardModel.Problems, onTemplate(pos, FaultTemplateSide,
			"this template has no "+missing+", and lays out nothing"))
		return
	}
	r.validateTemplatePlaceholders(pos, template)
}

func (r *cardModelReader) addTemplate(sec section, secsAfter []section) {
	pos := len(r.cardModel.Templates)
	template := CardTemplate{Name: sec.name}

	sides := findTemplateSides(secsAfter)
	end := sec.to
	if len(sides) > 0 {
		end = sides[len(sides)-1].to
	}

	has, opening := findTemplateSideOpenings(sides)

	leadTo := end
	if len(opening) > 0 {
		leadTo = sides[opening[0]].head
	}
	lead, leadEnd := readTrimmedRun(r.body, sec.from, leadTo)
	template.Preamble = lead
	r.read = getTrimmedEnd(r.body, sec.head, sec.from)
	if lead != "" {
		r.read = leadEnd
	}

	r.readTemplateSides(&template, sides, opening, end)
	r.cardModel.Templates = append(r.cardModel.Templates, template)
	r.validateTemplate(pos, has, template)
}

// ReadCardModel reads a note as a card model: the fields it declares in its
// frontmatter, and the templates its body lays out.
//
// It never fails. A template missing a side and a placeholder that is not
// declared are problems of the card model, and everything else in the file is read.
func ReadCardModel(n domain.Note) CardModel {
	body := []byte(n.Body)
	declaredFields, problems := fields(n.Frontmatter)
	declared := make(map[string]bool, len(declaredFields))
	for _, f := range declaredFields {
		declared[f] = true
	}

	r := cardModelReader{
		body:     body,
		declared: declared,
		read:     0,
		cardModel: CardModel{
			Ref:       n.FileStat,
			Fields:    declaredFields,
			Problems:  problems,
			Preamble:  "",
			Templates: nil,
			Tail:      "",
		},
	}

	secs := sections(body, 2, 3)
	firstTemplate := len(body)
	for _, sec := range secs {
		if sec.level == 2 {
			firstTemplate = sec.head
			break
		}
	}
	r.cardModel.Preamble = markdown.Normalise(string(body[:firstTemplate]))

	for i, sec := range secs {
		if sec.level != 2 {
			continue
		}
		r.addTemplate(sec, secs[i+1:])
	}

	if len(r.cardModel.Templates) > 0 {
		r.cardModel.Tail = markdown.Normalise(string(body[r.read:]))
	}
	return r.cardModel
}

// fields returns what the frontmatter declares, in the order the user supplies
// them. Names are compared as written, and the first of two with the same name is
// kept.
//
// An entry that is not a name is not a field, and every template that places it
// reports a problem. A card model with no field cuts nothing, and reports that.
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
			"this card model declares no field, so a card cut by it has nothing to be named by"))
	}
	return out, problems
}
