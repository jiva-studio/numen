// Package tabkind chooses the kind of tab a note opens in.
package tabkind

import (
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// Choose gives a note the kind of tab it opens in, from the tab kinds of the enabled plugins.
type Choose struct {
	kinds pluginport.PluginTabKinds
	types contenttypeport.ContentTypeTable
}

// NewChoose makes the choice over the tab kinds and the table that gives a file its type. Without
// tab kinds it chooses none.
func NewChoose(kinds pluginport.PluginTabKinds, types contenttypeport.ContentTypeTable) *Choose {
	return &Choose{kinds: kinds, types: types}
}

// Execute gives the kind of tab the note at the path opens in: of the kinds whose when condition
// holds over the note, a default kind before an alternate one, and among those the one
// domain.ChooseHandler picks, the lower plugin id first. A kind without a condition takes no note.
// Empty where no condition holds.
func (u *Choose) Execute(path string, noteType domain.NoteType, properties map[string]any) string {
	if u.kinds == nil {
		return ""
	}
	contentType := domain.UnknownContentType
	if u.types != nil {
		contentType = u.types.GetContentType(path)
	}
	values := pluginport.NewTargetValues(path, false, contentType).WithNote(noteType, properties).Get()
	var matching []domain.ContentHandler
	for _, one := range u.kinds.ListTabKinds() {
		if one.TabKind.When == "" {
			continue
		}
		condition, err := pluginport.ParseCondition(one.TabKind.When)
		if err != nil || condition.Evaluate(values) != pluginport.ConditionTrue {
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
