package manifest

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
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
	TabKinds         []TabKind
	Feature          string
	Category         string
	Themes           []Theme
	Readers          []Reader
	NoteTypes        []NoteType
	FrameOrigins     []string
	Commands         []Command
	Slots            []SlotEntry
	Handlers         []Handler
	Triggers         []Trigger
	TextKinds        []TextKind
	Listeners        []Listener
	Keybindings      []Keybinding
	ViewCapabilities []string
	DropHandlers     []DropHandler
	ConnectionTypes  []ConnectionType
}

// DropHandler lists the data types of a drag from outside the window that the plugin's frontend takes.
type DropHandler struct {
	Types []string
}

// TextKind is the readable name of one kind of text version the plugin makes.
type TextKind struct {
	Kind  string
	Title string
}

// GetTextKindTitle returns the name the plugin declares for the kind of text version, and false where
// it declares none.
func (c Contributes) GetTextKindTitle(kind string) (string, bool) {
	for _, declared := range c.TextKinds {
		if declared.Kind == kind {
			return declared.Title, true
		}
	}
	return "", false
}

// HasTabKind reports whether the plugin registers the kind of tab.
func (c Contributes) HasTabKind(kind string) bool {
	_, ok := c.GetTabKind(kind)
	return ok
}

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
	issues = append(issues, listTextKindIssues(c.TextKinds)...)
	issues = append(issues, m.listListenerIssues()...)
	issues = append(issues, m.listKeybindingIssues()...)
	issues = append(issues, m.listViewCapabilityIssues()...)
	issues = append(issues, m.listDropHandlerIssues()...)
	issues = append(issues, m.listConnectionTypeIssues()...)
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
			issues = append(
				issues,
				fmt.Errorf("%w: tab kind %q has icon %q, which is not lowercase words joined by hyphens", ErrInvalidContribution, declared.Kind, declared.Icon),
			)
		}
		if !declared.Rank.IsValid() {
			issues = append(
				issues,
				fmt.Errorf(
					"%w: tab kind %q has rank %q, expected %q or %q",
					ErrInvalidContribution,
					declared.Kind,
					declared.Rank,
					domain.RankDefault,
					domain.RankAlternate,
				),
			)
		}
		if _, err := when.Parse(declared.When); err != nil {
			issues = append(issues, fmt.Errorf("%w: tab kind %q: when: %w", ErrInvalidContribution, declared.Kind, err))
		}
	}
	return issues
}

func listTextKindIssues(kinds []TextKind) []error {
	var issues []error
	observed := map[string]bool{}
	for _, declared := range kinds {
		switch {
		case strings.TrimSpace(declared.Kind) == "":
			issues = append(issues, fmt.Errorf("%w: a text kind has no kind", ErrInvalidContribution))
		case observed[declared.Kind]:
			issues = append(issues, fmt.Errorf("%w: text kind %q is declared twice", ErrInvalidContribution, declared.Kind))
		}
		observed[declared.Kind] = true
		if strings.TrimSpace(declared.Title) == "" {
			issues = append(issues, fmt.Errorf("%w: text kind %q has no title", ErrInvalidContribution, declared.Kind))
		}
	}
	return issues
}

// GetApplicationViewCapabilities returns the kinds of view the application defines. A plugin draws one for
// others to mount only when its manifest lists the kind under contributes.viewCapabilities.
func GetApplicationViewCapabilities() []string {
	return []string{"conversation", "note-editor", "vault-summary"}
}

func (m Manifest) listViewCapabilityIssues() []error {
	var issues []error
	if len(m.Contributes.ViewCapabilities) > 0 && !m.HasFrontend() {
		issues = append(issues, fmt.Errorf("%w: the plugin declares view capabilities and has no frontend to register them", ErrInvalidContribution))
	}
	observed := map[string]bool{}
	for _, capability := range m.Contributes.ViewCapabilities {
		if observed[capability] {
			issues = append(issues, fmt.Errorf("%w: view capability %q is declared twice", ErrInvalidContribution, capability))
		}
		observed[capability] = true
		if !m.isViewCapabilityAllowed(capability) {
			issues = append(
				issues,
				fmt.Errorf(
					"%w: view capability %q is neither one of %s nor named %q followed by a name",
					ErrInvalidContribution,
					capability,
					strings.Join(GetApplicationViewCapabilities(), ", "),
					m.ID+".",
				),
			)
		}
	}
	return issues
}

func (m Manifest) isViewCapabilityAllowed(capability string) bool {
	if slices.Contains(GetApplicationViewCapabilities(), capability) {
		return true
	}
	rest, isOwn := strings.CutPrefix(capability, m.ID+".")
	return isOwn && rest != "" && tabKindPattern.MatchString(rest)
}

func (m Manifest) listDropHandlerIssues() []error {
	var issues []error
	if len(m.Contributes.DropHandlers) > 0 && !m.HasFrontend() {
		issues = append(issues, fmt.Errorf("%w: the plugin declares drop handlers and has no frontend to register them", ErrInvalidContribution))
	}
	observed := map[string]bool{}
	for _, handler := range m.Contributes.DropHandlers {
		if len(handler.Types) == 0 {
			issues = append(issues, fmt.Errorf("%w: a drop handler lists no types", ErrInvalidContribution))
		}
		for _, dropType := range handler.Types {
			switch {
			case strings.TrimSpace(dropType) == "" || dropType != strings.TrimSpace(dropType):
				issues = append(issues, fmt.Errorf("%w: drop type %q is empty or has surrounding space", ErrInvalidContribution, dropType))
			case observed[dropType]:
				issues = append(issues, fmt.Errorf("%w: drop type %q is declared twice", ErrInvalidContribution, dropType))
			}
			observed[dropType] = true
		}
	}
	return issues
}

// NoteType declares one type of note and how the index keeps the notes of it.
type NoteType struct {
	Type            string
	Chunks          *bool
	Headings        *[]int
	IDHeadingLevels []int
	IDPattern       string
}

func (m Manifest) listNoteTypeIssues() []error {
	var issues []error
	observed := map[string]bool{}
	for _, declared := range m.Contributes.NoteTypes {
		switch {
		case !manifestNoteTypePattern.MatchString(declared.Type):
			issues = append(issues, fmt.Errorf("%w: %q", ErrInvalidNoteType, declared.Type))
		case observed[declared.Type]:
			issues = append(issues, fmt.Errorf("%w: %q is declared twice", ErrInvalidNoteType, declared.Type))
		}
		observed[declared.Type] = true
		if _, err := regexp.Compile(declared.IDPattern); err != nil {
			issues = append(issues, fmt.Errorf("%w: %q has an id pattern that is no pattern: %w", ErrInvalidNoteType, declared.Type, err))
		}
		levels := declared.IDHeadingLevels
		if declared.Headings != nil {
			levels = append(slices.Clone(*declared.Headings), levels...)
		}
		for _, level := range levels {
			if level < 1 || level > 6 {
				issues = append(issues, fmt.Errorf("%w: %q names heading level %d", ErrInvalidNoteType, declared.Type, level))
			}
		}
	}
	return issues
}

// GetProfile is how the index keeps the notes of the type.
func (t NoteType) GetProfile() domain.NoteProfile {
	profile := domain.DefaultProfile()
	if t.Chunks != nil {
		profile.HasChunks = *t.Chunks
	}
	if t.Headings != nil {
		profile.HeadingLevels = slices.Clone(*t.Headings)
		if profile.HeadingLevels == nil {
			profile.HeadingLevels = []int{}
		}
	}
	profile.IDHeadingLevels = slices.Clone(t.IDHeadingLevels)
	if t.IDPattern != "" {
		// listNoteTypeIssues rejects a pattern that does not compile.
		if pattern, err := regexp.Compile("^(?:" + t.IDPattern + ")$"); err == nil {
			profile.IDPattern = pattern
		}
	}
	return profile
}

// GetNoteType is the declaration of a type of note the plugin gives meaning to.
func (m Manifest) GetNoteType(noteType string) (NoteType, bool) {
	for _, declared := range m.Contributes.NoteTypes {
		if declared.Type == noteType {
			return declared, true
		}
	}
	return NoteType{}, false
}
