package reasoning

import (
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"strings"
)

// Listed is a model as a program lists it.
type Listed struct {
	Name  string
	Title string
}

// Family is a model a person chooses, with the depths of reasoning it can work at. A model listed
// with no depth in its title is a family of no levels.
type Family struct {
	// Name is what a task asks for the model by.
	Name  string
	Title string
	// Levels run from the least reasoning to the most. None where the model offers no choice.
	Levels []Level
	// Default is the level the model works at where a task names none.
	Default string

	// variants is the name of the listed model by level. The family of no levels holds its own name
	// under the empty level.
	variants map[string]string
}

// ToProto is the family in the schema's words.
func (f Family) ToProto() *numenv1.AgentModel {
	return NewModel(f.Name, f.Title, f.Levels, f.Default)
}

// GetVariant is the name of the listed model that works at the level, the default level where the
// family has none of that name.
func (f Family) GetVariant(level string) string {
	if len(f.Levels) == 0 {
		return f.variants[""]
	}
	return f.variants[Normalize(f.Levels, level)]
}

// SplitTitle reads "Name (Level)" into the name and the level. A title whose parentheses hold no
// level name is returned whole, with no level.
func SplitTitle(title string) (string, string) {
	title = strings.TrimSpace(title)
	if !strings.HasSuffix(title, ")") {
		return title, ""
	}
	open := strings.LastIndex(title, "(")
	if open <= 0 {
		return title, ""
	}
	level := strings.TrimSpace(title[open+1 : len(title)-1])
	base := strings.TrimSpace(title[:open])
	if base == "" || !IsKnown(level) {
		return title, ""
	}
	return base, strings.ToLower(level)
}

// Fold gathers the models that differ by the level in their titles into one family each, in the order
// the first of each is listed. A title standing alone with a level in it stays as it was listed.
func Fold(models []Listed) []Family {
	type group struct {
		base   string
		listings []Listed
		levels []string
	}
	var groups []*group
	byBase := map[string]*group{}
	for _, one := range models {
		base, level := SplitTitle(one.Title)
		if level == "" {
			groups = append(groups, &group{base: one.Title, listed: []Listed{one}, levels: []string{""}})
			continue
		}
		found := byBase[base]
		if found == nil {
			found = &group{base: base}
			byBase[base] = found
			groups = append(groups, found)
		}
		found.listed = append(found.listed, one)
		found.levels = append(found.levels, level)
	}

	families := make([]Family, 0, len(groups))
	for _, one := range groups {
		if len(one.listed) == 1 {
			only := one.listed[0]
			families = append(families, Family{
				Name: only.Name, Title: only.Title, variants: map[string]string{"": only.Name},
			})
			continue
		}
		families = append(families, newFamily(one.base, one.listed, one.levels))
	}
	return families
}

// newFamily is the family of the models listed once per level.
func newFamily(title string, listings []Listed, names []string) Family {
	levels := NewLevels(names...)
	Sort(levels)
	variants := make(map[string]string, len(listed))
	for place, one := range listed {
		variants[names[place]] = one.Name
	}
	return Family{
		Name: getFamilyName(listings, names), Title: title, Levels: levels,
		Default: GetDefault(levels), variants: variants,
	}
}

// getFamilyName is the name the listed models share once the level is taken from the end of each,
// else the name of the first.
func getFamilyName(listings []Listed, levels []string) string {
	shared := ""
	for place, one := range listed {
		trimmed, found := strings.CutSuffix(one.Name, "-"+levels[place])
		if !found || trimmed == "" || (place > 0 && trimmed != shared) {
			return listed[0].Name
		}
		shared = trimmed
	}
	return shared
}

// ResolveModel is the model a program is started with for the family named and the level asked. A name
// that is no family's is returned as it is.
func ResolveModel(families []Family, model, level string) string {
	for _, one := range families {
		if one.Name == model {
			return one.GetVariant(level)
		}
	}
	return model
}
