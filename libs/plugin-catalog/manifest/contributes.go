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
	// Themes are the stylesheets the plugin adds to the list of themes.
	Themes []Theme
	// Handles are the capabilities the plugin takes part in for files and notes of some types.
	Handles []Handle
	// Readers are the tabs of the plugin's frontend that open files, each by the id the frontend
	// registers it under.
	Readers []Reader
	// NoteTypes are the types of note the plugin gives meaning to, and how the index keeps the notes of
	// each. A note of a type no plugin declares is kept as an ordinary note is.
	NoteTypes []NoteType
	// FrameOrigins are the https origins the plugin's pages frame, such as a video site's player. The
	// window's frame-src names them while the plugin is enabled.
	FrameOrigins []string
	// Commands are the commands the plugin declares, whichever part of the plugin runs them. The
	// palette and the command line list them; a command is run by the frontend when the frontend
	// registers a handler for it and by the backend otherwise.
	Commands []Command
	// Slots place the commands, views and submenus of the plugin in the slots of a window and in submenus.
	Slots []SlotEntry
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
	return TabKind{}, false
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
// the category are written as the application names them, and that a plugin registering tabs or views
// has a frontend to register them from.
func (m Manifest) listContributionIssues() []error {
	var issues []error
	c := m.Contributes
	issues = append(issues, m.listFrontendIssues()...)
	issues = append(issues, listTabKindIssues(c.TabKinds)...)
	if c.Category != "" && !categoryNamePattern.MatchString(c.Category) {
		issues = append(issues, fmt.Errorf("%w: category %q is not lowercase words joined by hyphens", ErrInvalidContribution, c.Category))
	}
	if c.Feature != strings.TrimSpace(c.Feature) {
		issues = append(issues, fmt.Errorf("%w: feature %q has surrounding space", ErrInvalidContribution, c.Feature))
	}
	issues = append(issues, c.listThemeIssues()...)
	issues = append(issues, m.listHandleIssues()...)
	issues = append(issues, m.listReaderIssues()...)
	issues = append(issues, m.listSlotIssues()...)
	return append(issues, m.listFrameIssues()...)
}

// listFrontendIssues checks that a plugin registering tabs or views has a frontend to register them from.
func (m Manifest) listFrontendIssues() []error {
	var issues []error
	if len(m.Contributes.TabKinds) > 0 && !m.HasFrontend() {
		issues = append(issues, fmt.Errorf("%w: the plugin declares tab kinds and has no frontend to register them", ErrInvalidContribution))
	}
	if m.Contributes.hasSlotViews() && !m.HasFrontend() {
		issues = append(issues, fmt.Errorf("%w: the plugin places views in slots and has no frontend to register them", ErrInvalidContribution))
	}
	return issues
}

// listTabKindIssues checks that each kind of tab has a unique id and a title and an icon written as the application names it.
func listTabKindIssues(kinds []TabKind) []error {
	var issues []error
	observed := map[string]bool{}
	for _, declared := range kinds {
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
	return issues
}
