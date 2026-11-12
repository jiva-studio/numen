package manifestjson

import (
	"errors"
	"fmt"
	"math"

	"google.golang.org/protobuf/types/known/structpb"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func newContributes(wire *v1.ManifestContributes) (manifest.Contributes, error) {
	var kinds []manifest.TabKind
	for _, one := range wire.GetTabKinds() {
		kinds = append(kinds, manifest.TabKind{
			Kind: one.GetKind(), Title: one.GetTitle(), Icon: one.GetIcon(), When: one.GetWhen(), Rank: newTabKindRank(one.GetRank()),
		})
	}
	var themes []manifest.Theme
	for _, one := range wire.GetThemes() {
		themes = append(themes, manifest.Theme{ID: one.GetId(), Name: one.GetName(), Appearance: one.GetAppearance(), File: one.GetFile()})
	}
	connectionTypes, err := newConnectionTypes(wire.GetConnectionTypes())
	if err != nil {
		return manifest.Contributes{}, err
	}
	noteTypes, err := newNoteTypes(wire.GetNoteTypes())
	if err != nil {
		return manifest.Contributes{}, err
	}
	return manifest.Contributes{
		TabKinds:     kinds,
		Feature:      wire.GetFeature(),
		Category:     wire.GetCategory(),
		Themes:       themes,
		Readers:      newReaders(wire.GetReaders()),
		NoteTypes:    noteTypes,
		FrameOrigins: wire.GetFrameOrigins(),
		Commands:     newCommands(wire.GetCommands()),
		Slots:        newSlots(wire.GetSlots()),
		Handlers:     newHandlers(wire.GetHandlers()),
		Triggers:     newTriggers(wire.GetTriggers()),
		TextKinds:    newTextKinds(wire.GetTextKinds()),
		Listeners:    newListeners(wire.GetListeners()),
		Keybindings:  newKeybindings(wire.GetKeybindings()),

		ViewCapabilities: wire.GetViewCapabilities(),
		DropHandlers:     newDropHandlers(wire.GetDropHandlers()),
		ConnectionTypes:  connectionTypes,
	}, nil
}

func newSlots(wire []*v1.ManifestSlotEntry) []manifest.SlotEntry {
	var out []manifest.SlotEntry
	for _, one := range wire {
		var order *int
		if one.Order != nil {
			value := int(one.GetOrder())
			order = &value
		}
		out = append(out, manifest.SlotEntry{
			Slot:    one.GetSlot(),
			Command: one.GetCommand(),
			View:    one.GetView(),
			Submenu: one.GetSubmenu(),
			Title:   one.GetTitle(),
			Icon:    one.GetIcon(),
			Group:   one.GetGroup(),
			Order:   order,
			When:    one.GetWhen(),
			Values:  one.GetValues(),
		})
	}
	return out
}

func newKeybindings(wire []*v1.ManifestKeybinding) []manifest.Keybinding {
	var out []manifest.Keybinding
	for _, one := range wire {
		out = append(out, manifest.Keybinding{Key: one.GetKey(), Command: one.GetCommand(), Values: one.GetValues(), When: one.GetWhen()})
	}
	return out
}

var errHeadingShape = errors.New("headings must be a list of whole numbers")

func newNoteTypes(wire []*v1.ManifestNoteType) ([]manifest.NoteType, error) {
	var out []manifest.NoteType
	for _, one := range wire {
		headings, err := newHeadings(one.GetHeadings())
		if err != nil {
			return nil, fmt.Errorf("note type %q: %w", one.GetType(), err)
		}
		var levels []int
		for _, level := range one.GetIdHeadingLevels() {
			levels = append(levels, int(level))
		}
		out = append(out, manifest.NoteType{
			Type:            one.GetType(),
			Chunks:          one.Chunks,
			Headings:        headings,
			IDHeadingLevels: levels,
			IDPattern:       one.GetIdPattern(),
		})
	}
	return out, nil
}

func newHeadings(list *structpb.ListValue) (*[]int, error) {
	if list == nil {
		return nil, nil
	}
	levels := []int{}
	for _, item := range list.GetValues() {
		number, isNumber := item.GetKind().(*structpb.Value_NumberValue)
		if !isNumber || number.NumberValue != math.Trunc(number.NumberValue) {
			return nil, errHeadingShape
		}
		levels = append(levels, int(number.NumberValue))
	}
	return &levels, nil
}

func newCommands(wire []*v1.ManifestCommand) []manifest.Command {
	var out []manifest.Command
	for _, one := range wire {
		out = append(out, manifest.Command{
			ID:         one.GetId(),
			Title:      one.GetTitle(),
			Icon:       one.GetIcon(),
			When:       one.GetWhen(),
			Permission: one.GetPermission(),
			Arguments:  newCommandArguments(one.GetArguments()),
		})
	}
	return out
}

func newCommandArguments(wire []*v1.ManifestCommandArgument) []manifest.CommandArgument {
	var out []manifest.CommandArgument
	for _, one := range wire {
		out = append(out, manifest.CommandArgument{
			Name:       one.GetName(),
			Title:      one.GetTitle(),
			Type:       one.GetType(),
			IsRequired: one.GetIsRequired(),
		})
	}
	return out
}

func newHandlers(wire []*v1.ManifestHandler) []manifest.Handler {
	var out []manifest.Handler
	for _, one := range wire {
		out = append(out, manifest.Handler{Command: one.GetCommand(), Title: one.GetTitle(), When: one.GetWhen()})
	}
	return out
}

func newTriggers(wire []*v1.ManifestTrigger) []manifest.Trigger {
	var out []manifest.Trigger
	for _, one := range wire {
		out = append(out, manifest.Trigger{Event: one.GetEvent(), Command: one.GetCommand(), When: one.GetWhen()})
	}
	return out
}

func newTextKinds(wire []*v1.ManifestTextKind) []manifest.TextKind {
	var out []manifest.TextKind
	for _, one := range wire {
		out = append(out, manifest.TextKind{Kind: one.GetKind(), Title: one.GetTitle()})
	}
	return out
}

func newReaders(wire []*v1.ManifestReader) []manifest.Reader {
	var out []manifest.Reader
	for _, one := range wire {
		out = append(out, manifest.Reader{ID: one.GetId(), ContentTypes: newContentTypes(one.GetContentTypes()), Rank: newHandlerRank(one.GetRank())})
	}
	return out
}

func newListeners(wire []*v1.ManifestListener) []manifest.Listener {
	var out []manifest.Listener
	for _, one := range wire {
		out = append(out, manifest.Listener{
			Name:        one.GetName(),
			Port:        int(one.GetPort()),
			PortSetting: one.GetPortSetting(),
			Protocol:    one.GetProtocol(),
		})
	}
	return out
}

func newDropHandlers(wire []*v1.ManifestDropHandler) []manifest.DropHandler {
	var out []manifest.DropHandler
	for _, one := range wire {
		out = append(out, manifest.DropHandler{Types: one.GetTypes()})
	}
	return out
}
