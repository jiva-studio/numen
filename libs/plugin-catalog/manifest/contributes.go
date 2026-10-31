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

// FileType is a type of file the plugin opens.
type FileType struct {
	// TabKind is the kind of tab that opens the file. It is one of the kinds the plugin registers.
	TabKind string
	// Extensions are the extensions, lowercase and without the dot, of the files of the type.
	Extensions []string
	// FileNames are the names, lowercase, of the files of the type whatever their extension.
	FileNames []string
}

// Contributes is what a plugin adds to the application, known before the frontend of the plugin is loaded.
type Contributes struct {
	// TabKinds are the kinds of tab the plugin registers.
	TabKinds []TabKind
	// Feature names what the plugin provides, in the words a placeholder for an unavailable tab uses.
	Feature string
	// Category groups the plugin in lists. Empty is a plugin in no category.
	Category string
	// FileTypes are the types of file the plugin opens.
	FileTypes []FileType
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
	issues = append(issues, c.listFileTypeIssues()...)
	if c.Feature != strings.TrimSpace(c.Feature) {
		issues = append(issues, fmt.Errorf("%w: feature %q has surrounding space", ErrInvalidContribution, c.Feature))
	}
	return issues
}

// listFileTypeIssues checks that each type of file names a kind of tab the plugin registers and at
// least one lowercase extension or file name, and that no extension or file name belongs to two types.
func (c Contributes) listFileTypeIssues() []error {
	var issues []error
	observed := map[string]bool{}
	for _, declared := range c.FileTypes {
		if !c.HasTabKind(declared.TabKind) {
			issues = append(issues, fmt.Errorf("%w: file type names the tab kind %q, which the plugin does not declare", ErrInvalidContribution, declared.TabKind))
		}
		if len(declared.Extensions)+len(declared.FileNames) == 0 {
			issues = append(issues, fmt.Errorf("%w: file type of tab kind %q names no extension or file name", ErrInvalidContribution, declared.TabKind))
		}
		for _, extension := range declared.Extensions {
			issues = append(issues, listFileKeyIssues("extension", extension, "ext:"+extension, observed)...)
			if strings.Contains(extension, ".") {
				issues = append(issues, fmt.Errorf("%w: extension %q has a dot", ErrInvalidContribution, extension))
			}
		}
		for _, name := range declared.FileNames {
			issues = append(issues, listFileKeyIssues("file name", name, "name:"+name, observed)...)
		}
	}
	return issues
}

func listFileKeyIssues(noun, value, key string, observed map[string]bool) []error {
	var issues []error
	if value == "" || value != strings.ToLower(strings.TrimSpace(value)) {
		issues = append(issues, fmt.Errorf("%w: %s %q is not lowercase text without surrounding space", ErrInvalidContribution, noun, value))
	}
	if observed[key] {
		issues = append(issues, fmt.Errorf("%w: %s %q is declared twice", ErrInvalidContribution, noun, value))
	}
	observed[key] = true
	return issues
}
