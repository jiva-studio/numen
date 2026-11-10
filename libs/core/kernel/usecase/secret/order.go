package secret

import (
	"slices"
	"sort"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Section is the secrets of one group in list order. IsAsked is true when an installed plugin
// declares the group.
type Section struct {
	Group   string
	IsAsked bool
	Secrets []plugin.Secret
}

// Menu is the secrets the menu of one declaration offers, in order.
type Menu struct {
	Declaration plugin.DeclarationKey
	Secrets     []plugin.Secret
}

// ListAskedGroups returns the groups declared by plugins, once each, in byte order.
func ListAskedGroups(declarations []plugin.DeclarationKey) []string {
	var groups []string
	for _, one := range declarations {
		if one.Group != "" && !slices.Contains(groups, one.Group) {
			groups = append(groups, one.Group)
		}
	}
	sort.Strings(groups)
	return groups
}

// ListSections returns the secret sections in page order: declared groups first, then custom
// groups, sorted by name byte by byte.
func ListSections(list []plugin.Secret, requested []string) []Section {
	var others []string
	for _, one := range list {
		if !slices.Contains(requested, one.Group) && !slices.Contains(others, one.Group) {
			others = append(others, one.Group)
		}
	}
	sort.Strings(others)
	sections := make([]Section, 0, len(requested)+len(others))
	for _, group := range slices.Concat(requested, others) {
		section := Section{Group: group, IsAsked: slices.Contains(requested, group), Secrets: []plugin.Secret{}}
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
func ListInPageOrder(list []plugin.Secret, requested []string) []plugin.Secret {
	ordered := make([]plugin.Secret, 0, len(list))
	for _, section := range ListSections(list, requested) {
		ordered = append(ordered, section.Secrets...)
	}
	return ordered
}

// ListMenu returns the secrets the menu of a declaration offers: those of the declared group first
// where it has one, then the others, each in the order of the page.
func ListMenu(page []Section, declaration plugin.DeclarationKey) []plugin.Secret {
	var first, rest []plugin.Secret
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
func ListAllowed(page []plugin.Secret, declaration plugin.DeclarationKey) []plugin.Secret {
	var first, rest []plugin.Secret
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
