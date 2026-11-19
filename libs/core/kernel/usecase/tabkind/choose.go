// Package tabkind chooses the kind of tab a note opens in.
package tabkind

import (
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// Choose gives a note the kind of tab it opens in, from the tab kinds of the enabled plugins.
type Choose struct {
	kinds      pluginport.PluginTabKinds
	types      contenttypeport.ContentTypeTable
	config     configValue
	conditions *when.Conditions
}

// NewChoose makes the choice over the tab kinds, the table that gives a file its type, and the
// settings a condition reads as config, which may be none. It needs the tab kinds and the table.
func NewChoose(kinds pluginport.PluginTabKinds, types contenttypeport.ContentTypeTable, config pluginport.WhenConfig) *Choose {
	var value configValue = noConfig{}
	if config != nil {
		value = givenConfig{config}
	}
	return &Choose{kinds: kinds, types: types, config: value, conditions: when.NewConditions()}
}

type parsedKind struct {
	entry     plugin.TabKindEntry
	condition when.Condition
	hasConfig bool
}

// Execute gives the kind of tab the note at the path opens in: of the kinds whose when condition
// holds over the note, a default kind before an alternate one, and among those the one
// domain.ChooseHandler picks, the lower plugin id first. A kind without a condition takes no note.
// Empty where no condition holds.
func (u *Choose) Execute(path string, noteType domain.NoteType, properties map[string]any) string {
	return u.chooseForNote(u.parseTabKinds(), nil, path, noteType, properties)
}

// ChooseMany gives the kind of tab each note in heads opens in, reading the tab kinds of the enabled
// plugins once for the batch. A path where no condition holds is omitted.
func (u *Choose) ChooseMany(heads map[string]frontmatter.Head) map[string]string {
	out := make(map[string]string, len(heads))
	if len(heads) == 0 {
		return out
	}
	kinds := u.parseTabKinds()
	if len(kinds) == 0 {
		return out
	}
	var cachedConfig map[string]any
	for path, head := range heads {
		if kind := u.chooseForNote(kinds, &cachedConfig, path, head.Type, head.Values); kind != "" {
			out[path] = kind
		}
	}
	return out
}

func (u *Choose) parseTabKinds() []parsedKind {
	entries := u.kinds.ListTabKinds()
	out := make([]parsedKind, 0, len(entries))
	for _, one := range entries {
		if one.TabKind.When == "" {
			continue
		}
		condition, err := u.conditions.Parse(one.TabKind.When)
		if err != nil {
			continue
		}
		out = append(out, parsedKind{
			entry:     one,
			condition: condition,
			hasConfig: slices.Contains(condition.ListNames(), "config"),
		})
	}
	return out
}

func (u *Choose) chooseForNote(
	kinds []parsedKind, cachedConfig *map[string]any, path string, noteType domain.NoteType, properties map[string]any,
) string {
	contentType := u.types.GetContentType(path)
	values := plugin.NewTargetValues(path, false, contentType).WithNote(noteType, properties).Get()
	var matching []domain.ContentHandler
	for _, one := range kinds {
		u.addConfig(values, cachedConfig, one.hasConfig)
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

func (u *Choose) addConfig(values map[string]any, cachedConfig *map[string]any, needsConfig bool) {
	if !needsConfig {
		return
	}
	if _, isRead := values["config"]; isRead {
		return
	}
	if cachedConfig == nil {
		u.config.add(values)
		return
	}
	if *cachedConfig == nil {
		*cachedConfig = u.getConfig()
	}
	if *cachedConfig != nil {
		values["config"] = *cachedConfig
	}
}

func (u *Choose) getConfig() map[string]any {
	values := map[string]any{}
	u.config.add(values)
	config, ok := values["config"].(map[string]any)
	if !ok {
		return nil
	}
	return config
}

// configValue gives a condition the config variable.
type configValue interface {
	add(values map[string]any)
}

type givenConfig struct{ config pluginport.WhenConfig }

func (c givenConfig) add(values map[string]any) { values["config"] = c.config.GetWhenConfig() }

// noConfig leaves a condition over config undecided.
type noConfig struct{}

func (noConfig) add(map[string]any) {}
