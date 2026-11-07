package secret

import (
	"slices"
	"sort"
)

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

// ListInPageOrder returns the secrets in the order of the page of secrets: the sections of the
// groups an installed plugin asks for by name, then the sections of the groups the person made by
// name, and inside a section the order of the list. Names are compared byte by byte.
func ListInPageOrder(list []Secret, requested []string) []Secret {
	var others []string
	for _, one := range list {
		if !slices.Contains(requested, one.Group) && !slices.Contains(others, one.Group) {
			others = append(others, one.Group)
		}
	}
	sort.Strings(others)
	ordered := make([]Secret, 0, len(list))
	for _, group := range slices.Concat(requested, others) {
		for _, one := range list {
			if one.Group == group {
				ordered = append(ordered, one)
			}
		}
	}
	return ordered
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
