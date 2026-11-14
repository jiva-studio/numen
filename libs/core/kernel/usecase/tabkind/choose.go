// Package tabkind chooses the kind of tab a note opens in.
package tabkind

import (
	"slices"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/when"

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

// Execute gives the kind of tab the note at the path opens in: of the kinds whose when condition
// holds over the note, a default kind before an alternate one, and among those the one
// domain.ChooseHandler picks, the lower plugin id first. A kind without a condition takes no note.
// Empty where no condition holds.
func (u *Choose) Execute(path string, noteType domain.NoteType, properties map[string]any) string {
	contentType := u.types.GetContentType(path)
	values := plugin.NewTargetValues(path, false, contentType).WithNote(noteType, properties).Get()
	var matching []domain.ContentHandler
	for _, one := range u.kinds.ListTabKinds() {
		if one.TabKind.When == "" {
			continue
		}
		condition, err := u.conditions.Parse(one.TabKind.When)
		if err != nil {
			continue
		}
		if _, isRead := values["config"]; !isRead && slices.Contains(condition.ListNames(), "config") {
			u.config.add(values)
		}
		if condition.Evaluate(values) != when.True {
			continue
		}
		matching = append(matching, domain.ContentHandler{
			PluginID: one.PluginID, ID: one.TabKind.Kind, Type: domain.AnyContentType, Rank: one.TabKind.Rank,
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

// configValue gives a condition the config variable.
type configValue interface {
	add(values map[string]any)
}

type givenConfig struct{ config pluginport.WhenConfig }

func (c givenConfig) add(values map[string]any) { values["config"] = c.config.GetWhenConfig() }

// noConfig leaves a condition over config undecided.
type noConfig struct{}

func (noConfig) add(map[string]any) {}
