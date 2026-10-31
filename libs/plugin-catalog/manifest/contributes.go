package manifest

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrInvalidContribution is a contribution a manifest declares wrongly.
	ErrInvalidContribution = errors.New("invalid contribution")

	tabKindPattern      = regexp.MustCompile(`^[a-z][a-z0-9]*([.:-][a-z0-9]+)*$`)
	iconNamePattern     = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	categoryNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
)

// TabKind is a kind of tab the plugin's frontend registers.
type TabKind struct {
	// Kind is the id the frontend registers the kind under.
	Kind string
	// Title is the name of the kind that a tab of it is shown with.
	Title string
	// Icon is the kebab-case name of an icon of the application's icon set. Empty is a kind without one.
	Icon string
}

// Contributes is what a plugin adds to the application, known before the frontend of the plugin is loaded.
type Contributes struct {
	// TabKinds are the kinds of tab the plugin registers.
	TabKinds []TabKind
	// Feature names what the plugin provides, in the words a placeholder for an unavailable tab uses.
	Feature string
	// Category groups the plugin in lists. Empty is a plugin in no category.
	Category string
}

// HasTabKind reports whether the plugin registers the kind of tab.
func (c Contributes) HasTabKind(kind string) bool {
	_, ok := c.GetTabKind(kind)
	return ok
}

// GetTabKind returns the declaration of the kind of tab.
func (c Contributes) GetTabKind(kind string) (TabKind, bool) {
	for _, declared := range c.TabKinds {
		if declared.Kind == kind {
			return declared, true
		}
	}
	return TabKind{Kind: "", Title: "", Icon: ""}, false
}

// ListTabKinds returns the ids of the kinds of tab the plugin registers.
func (c Contributes) ListTabKinds() []string {
	kinds := make([]string, 0, len(c.TabKinds))
	for _, declared := range c.TabKinds {
		kinds = append(kinds, declared.Kind)
	}
	return kinds
}

// listContributionIssues checks that each kind of tab has a unique id and a title, that the icons and
// the category are written as the application names them, and that a plugin registering tabs has a
// frontend to register them from.
func (m Manifest) listContributionIssues() []error {
	var issues []error
	c := m.Contributes
	if len(c.TabKinds) > 0 && !m.HasFrontend() {
		issues = append(issues, fmt.Errorf("%w: the plugin declares tab kinds and has no frontend to register them", ErrInvalidContribution))
	}
	observed := map[string]bool{}
	for _, declared := range c.TabKinds {
		switch {
		case !tabKindPattern.MatchString(declared.Kind):
			issues = append(issues, fmt.Errorf("%w: tab kind %q is not lowercase words", ErrInvalidContribution, declared.Kind))
		case observed[declared.Kind]:
			issues = append(issues, fmt.Errorf("%w: tab kind %q is declared twice", ErrInvalidContribution, declared.Kind))
		}
		observed[declared.Kind] = true
		if strings.TrimSpace(declared.Title) == "" {
			issues = append(issues, fmt.Errorf("%w: tab kind %q has no title", ErrInvalidContribution, declared.Kind))
		}
		if declared.Icon != "" && !iconNamePattern.MatchString(declared.Icon) {
			issues = append(issues, fmt.Errorf("%w: tab kind %q has icon %q, which is not lowercase words joined by hyphens", ErrInvalidContribution, declared.Kind, declared.Icon))
		}
	}
	if c.Category != "" && !categoryNamePattern.MatchString(c.Category) {
		issues = append(issues, fmt.Errorf("%w: category %q is not lowercase words joined by hyphens", ErrInvalidContribution, c.Category))
	}
	if c.Feature != strings.TrimSpace(c.Feature) {
		issues = append(issues, fmt.Errorf("%w: feature %q has surrounding space", ErrInvalidContribution, c.Feature))
	}
	return issues
}
