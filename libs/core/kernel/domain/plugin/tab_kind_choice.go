package plugin

import (
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
)

// TabKindChoice gives a note the kind of tab it opens in, from the tab kinds of the enabled plugins.
type TabKindChoice struct {
	listKinds      func() []TabKindEntry
	getContentType func(path string) domain.ContentType
	getConfig      func() map[string]any
	conditions     *when.Conditions
}

// NewTabKindChoice makes the choice over the tab kinds the function lists, with the function that gives a
// file its content type and the function that gives the settings as the config variable of a condition.
// A nil config leaves a condition over config undecided.
func NewTabKindChoice(
	listKinds func() []TabKindEntry, getContentType func(path string) domain.ContentType, getConfig func() map[string]any,
) *TabKindChoice {
	return &TabKindChoice{listKinds: listKinds, getContentType: getContentType, getConfig: getConfig, conditions: when.NewConditions()}
}

type parsedTabKind struct {
	entry     TabKindEntry
	condition when.Condition
	hasConfig bool
}

// Choose gives the kind of tab the note at the path opens in: of the kinds whose when condition
// holds over the note, a default kind before an alternate one, and among those the one
// domain.ChooseHandler picks, the lower plugin id first. A kind without a condition takes no note.
// Empty where no condition holds.
func (c *TabKindChoice) Choose(path string, noteType domain.NoteType, properties map[string]any) string {
	return c.chooseForNote(c.parseTabKinds(), nil, path, noteType, properties)
}

// ChooseMany gives the kind of tab each note in heads opens in, reading the tab kinds of the enabled
// plugins once for the batch. A path where no condition holds is omitted.
func (c *TabKindChoice) ChooseMany(heads map[string]frontmatter.Head) map[string]string {
	out := make(map[string]string, len(heads))
	if len(heads) == 0 {
		return out
	}
	kinds := c.parseTabKinds()
	if len(kinds) == 0 {
		return out
	}
	var cachedConfig map[string]any
	for path, head := range heads {
		if kind := c.chooseForNote(kinds, &cachedConfig, path, head.Type, head.Values); kind != "" {
			out[path] = kind
		}
	}
	return out
}

func (c *TabKindChoice) parseTabKinds() []parsedTabKind {
	entries := c.listKinds()
	out := make([]parsedTabKind, 0, len(entries))
	for _, one := range entries {
		if one.TabKind.When == "" {
			continue
		}
		condition, err := c.conditions.Parse(one.TabKind.When)
		if err != nil {
			continue
		}
		out = append(out, parsedTabKind{
			entry:     one,
			condition: condition,
			hasConfig: slices.Contains(condition.ListNames(), "config"),
		})
	}
	return out
}

func (c *TabKindChoice) chooseForNote(
	kinds []parsedTabKind, cachedConfig *map[string]any, path string, noteType domain.NoteType, properties map[string]any,
) string {
	contentType := c.getContentType(path)
	values := NewTargetValues(path, false, contentType).WithNote(noteType, properties).Get()
	var matching []domain.ContentHandler
	for _, one := range kinds {
		c.addConfig(values, cachedConfig, one.hasConfig)
		if one.condition.Evaluate(values) != when.True {
			continue
		}
		matching = append(matching, domain.ContentHandler{
			PluginID: one.entry.PluginID, ID: one.entry.TabKind.Kind, Type: domain.AnyContentType, Rank: one.entry.TabKind.Rank,
		})
	}
	if chosen, ok := domain.ChooseHandler(contentType, matching); ok {
		return chosen.ID
	}
	for index := range matching {
		matching[index].Rank = domain.RankDefault
	}
	chosen, _ := domain.ChooseHandler(contentType, matching)
	return chosen.ID
}

func (c *TabKindChoice) addConfig(values map[string]any, cachedConfig *map[string]any, needsConfig bool) {
	if !needsConfig || c.getConfig == nil {
		return
	}
	if _, isRead := values["config"]; isRead {
		return
	}
	if cachedConfig == nil {
		values["config"] = c.getConfig()
		return
	}
	if *cachedConfig == nil {
		*cachedConfig = c.getConfig()
	}
	if *cachedConfig != nil {
		values["config"] = *cachedConfig
	}
}
