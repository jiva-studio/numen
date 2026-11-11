package plugin

import (
	"slices"
	"sort"
)

// SecretSection is the secrets of one group in list order. IsAsked is true when an installed plugin
// declares the group.
type SecretSection struct {
	Group   string
	IsAsked bool
	Secrets []Secret
}

// SecretMenu is the secrets the menu of one declaration offers, in order.
type SecretMenu struct {
	Declaration DeclarationKey
	Secrets     []Secret
}

// ListAskedSecretGroups returns the groups declared by plugins, once each, in byte order.
func ListAskedSecretGroups(declarations []DeclarationKey) []string {
	var groups []string
	for _, one := range declarations {
		if one.Group != "" && !slices.Contains(groups, one.Group) {
			groups = append(groups, one.Group)
		}
	}
	sort.Strings(groups)
	return groups
}

// ListSecretSections returns the secret sections in page order: declared groups first, then custom
// groups, sorted by name byte by byte.
func ListSecretSections(list []Secret, requested []string) []SecretSection {
	var others []string
	for _, one := range list {
		if !slices.Contains(requested, one.Group) && !slices.Contains(others, one.Group) {
			others = append(others, one.Group)
		}
	}
	sort.Strings(others)
	sections := make([]SecretSection, 0, len(requested)+len(others))
	for _, group := range slices.Concat(requested, others) {
		section := SecretSection{Group: group, IsAsked: slices.Contains(requested, group), Secrets: []Secret{}}
		for _, one := range list {
			if one.Group == group {
				section.Secrets = append(section.Secrets, one)
			}
		}
		sections = append(sections, section)
	}
	return sections
}

// ListSecretsInPageOrder returns the secrets in the order of the page of secrets, the secrets of
// ListSecretSections one after the other.
func ListSecretsInPageOrder(list []Secret, requested []string) []Secret {
	ordered := make([]Secret, 0, len(list))
	for _, section := range ListSecretSections(list, requested) {
		ordered = append(ordered, section.Secrets...)
	}
	return ordered
}

// ListSecretMenu returns the secrets the menu of a declaration offers: those of the declared group first
// where it has one, then the others, each in the order of the page.
func ListSecretMenu(page []SecretSection, declaration DeclarationKey) []Secret {
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

// ListAllowedSecrets returns, from secrets in the order of the page, those allowed to the declaration in
// the order it receives them: the secrets of the declared group first where it has one, then the
// others.
func ListAllowedSecrets(page []Secret, declaration DeclarationKey) []Secret {
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

// SecretListing is the secrets as the window shows them: the sections of the page in order, the menu of
// every declaration, and where the values are kept when it is a file. No value and no flag about a
// value is part of it, and the key store is not read.
type SecretListing struct {
	Sections        []SecretSection
	Menus           []SecretMenu
	CredentialsPath string
}
