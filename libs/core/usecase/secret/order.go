package secret

import (
	"slices"
	"sort"
)

// Section is the secrets of one group in the order of the list. IsAsked is true for a group an
// installed plugin asks for by name.
type Section struct {
	Group   string
	IsAsked bool
	Secrets []Secret
}

// Menu is the secrets the menu of one declaration offers, in order.
type Menu struct {
	Declaration Permission
	Secrets     []Secret
}

// ListAskedGroups returns the groups the declarations ask for, once each, in byte order.
func ListAskedGroups(declarations []Permission) []string {
	var groups []string
	for _, one := range declarations {
		if one.Group != "" && !slices.Contains(groups, one.Group) {
			groups = append(groups, one.Group)
		}
	}
	sort.Strings(groups)
	return groups
}

// ListSections returns the sections of the page of secrets: the groups an installed plugin asks
// for by name, each a section while it holds no secret, then the groups the person made by name,
// and inside a section the order of the list. Names are compared byte by byte.
func ListSections(list []Secret, requested []string) []Section {
	var others []string
	for _, one := range list {
		if !slices.Contains(requested, one.Group) && !slices.Contains(others, one.Group) {
			others = append(others, one.Group)
		}
	}
	sort.Strings(others)
	sections := make([]Section, 0, len(requested)+len(others))
	for _, group := range slices.Concat(requested, others) {
		section := Section{Group: group, IsAsked: slices.Contains(requested, group), Secrets: []Secret{}}
		for _, one := range list {
			if one.Group == group {
				section.Secrets = append(section.Secrets, one)
			}
		}
		sections = append(sections, section)
	}
	return sections
}

// ListInPageOrder returns the secrets in the order of the page of secrets, the secrets of
// ListSections one after the other.
func ListInPageOrder(list []Secret, requested []string) []Secret {
	ordered := make([]Secret, 0, len(list))
	for _, section := range ListSections(list, requested) {
		ordered = append(ordered, section.Secrets...)
	}
	return ordered
}

// ListMenu returns the secrets the menu of a declaration offers: those of the declared group first
// where it has one, then the others, each in the order of the page.
func ListMenu(page []Section, declaration Permission) []Secret {
	var first, rest []Secret
	for _, section := range page {
		if declaration.Group != "" && section.Group == declaration.Group {
			first = append(first, section.Secrets...)
		} else {
			rest = append(rest, section.Secrets...)
		}
	}
	return slices.Concat(first, rest)
}

// ListAllowed returns, from secrets in the order of the page, those allowed to the declaration in
// the order it receives them: the secrets of the declared group first where it has one, then the
// others.
func ListAllowed(page []Secret, declaration Permission) []Secret {
	var first, rest []Secret
	for _, one := range page {
		if !slices.Contains(one.Allowed, declaration) {
			continue
		}
		if declaration.Group != "" && one.Group == declaration.Group {
			first = append(first, one)
		} else {
			rest = append(rest, one)
		}
	}
	return slices.Concat(first, rest)
}
