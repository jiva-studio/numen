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
	Kind  string
	Title string
	Icon  string
}

// StatusView is a component of the plugin's frontend that the window draws in its status bar.
type StatusView struct {
	ID string
}

// Contributes is what a plugin adds to the application, known before the frontend of the plugin is loaded.
type Contributes struct {
	TabKinds     []TabKind
	Feature      string
	Category     string
	Themes       []Theme
	StatusViews  []StatusView
	Handles      []Handle
	Readers      []Reader
	NoteTypes    []NoteType
	FrameOrigins []string
	Commands     []Command
	Menus        []Menu
	Listeners    []Listener
}

// ListStatusViews returns the ids of the components the plugin registers for the status bar.
func (c Contributes) ListStatusViews() []string {
	ids := make([]string, 0, len(c.StatusViews))
	for _, declared := range c.StatusViews {
		ids = append(ids, declared.ID)
	}
	return ids
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

func (m Manifest) listContributionIssues() []error {
	var issues []error
	c := m.Contributes
	issues = append(issues, m.listFrontendIssues()...)
	issues = append(issues, listStatusViewIssues(c.StatusViews)...)
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
	issues = append(issues, m.listMenuIssues()...)
	issues = append(issues, m.listListenerIssues()...)
	return append(issues, m.listFrameIssues()...)
}

func (m Manifest) listFrontendIssues() []error {
	var issues []error
	if len(m.Contributes.TabKinds) > 0 && !m.HasFrontend() {
		issues = append(issues, fmt.Errorf("%w: the plugin declares tab kinds and has no frontend to register them", ErrInvalidContribution))
	}
	if len(m.Contributes.StatusViews) > 0 && !m.HasFrontend() {
		issues = append(issues, fmt.Errorf("%w: the plugin declares status views and has no frontend to register them", ErrInvalidContribution))
	}
	return issues
}

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

func listStatusViewIssues(views []StatusView) []error {
	var issues []error
	observed := map[string]bool{}
	for _, declared := range views {
		switch {
		case !tabKindPattern.MatchString(declared.ID):
			issues = append(issues, fmt.Errorf("%w: status view %q is not lowercase words", ErrInvalidContribution, declared.ID))
		case observed[declared.ID]:
			issues = append(issues, fmt.Errorf("%w: status view %q is declared twice", ErrInvalidContribution, declared.ID))
		}
		observed[declared.ID] = true
	}
	return issues
}
