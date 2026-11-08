package manifest

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-catalog/when"
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
	When  string
	Rank  domain.HandlerRank
}

// Contributes is what a plugin adds to the application, known before the frontend of the plugin is loaded.
type Contributes struct {
	TabKinds     []TabKind
	Feature      string
	Category     string
	Themes       []Theme
	Readers      []Reader
	NoteTypes    []NoteType
	FrameOrigins []string
	Commands     []Command
	Slots        []SlotEntry
	Handlers     []Handler
	Triggers     []Trigger
	Listeners    []Listener
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
	issues = append(issues, listTabKindIssues(c.TabKinds)...)
	if c.Category != "" && !categoryNamePattern.MatchString(c.Category) {
		issues = append(issues, fmt.Errorf("%w: category %q is not lowercase words joined by hyphens", ErrInvalidContribution, c.Category))
	}
	if c.Feature != strings.TrimSpace(c.Feature) {
		issues = append(issues, fmt.Errorf("%w: feature %q has surrounding space", ErrInvalidContribution, c.Feature))
	}
	issues = append(issues, c.listThemeIssues()...)
	issues = append(issues, m.listReaderIssues()...)
	issues = append(issues, m.listSlotIssues()...)
	issues = append(issues, m.listHandlerIssues()...)
	issues = append(issues, m.listTriggerIssues()...)
	issues = append(issues, m.listListenerIssues()...)
	return append(issues, m.listFrameIssues()...)
}

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
		if !declared.Rank.IsValid() {
			issues = append(issues, fmt.Errorf("%w: tab kind %q has rank %q, expected %q or %q", ErrInvalidContribution, declared.Kind, declared.Rank, domain.RankDefault, domain.RankAlternate))
		}
		if _, err := when.Parse(declared.When); err != nil {
			issues = append(issues, fmt.Errorf("%w: tab kind %q: when: %w", ErrInvalidContribution, declared.Kind, err))
		}
	}
	return issues
}
