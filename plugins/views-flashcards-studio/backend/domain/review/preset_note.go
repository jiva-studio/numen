package review

import (
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
)

// PresetLinkType is what a deck's link to its preset carries under `type`.
const PresetLinkType = "preset"

// The frontmatter keys of the two settings a goal is steered by.
const (
	MinutesADayKey = "minutes_a_day"
	RetentionKey   = "retention"
)

// ErrOutOfBounds is a setting outside what it may be. Nothing is written, and
// the settings are weighed before the file is opened.
var ErrOutOfBounds = errors.New("this setting is outside what a preset may hold")

// PresetNote is one preset as a read of its note found it.
type PresetNote struct {
	Path       string
	Outcome    edit.ReadOutcome
	Type       domain.NoteType
	Settings   Preset
	Stops      StopReason
	StopsToday StopReason
	Problems   []string
	FileStat   domain.FileStat
}

// CreateDefaultPresetNote returns the preset of a deck scheduled by no preset.
func CreateDefaultPresetNote() PresetNote {
	return PresetNote{
		Path: "", Outcome: edit.Ok, Type: "", Settings: NewDefaultPreset(), Stops: "", StopsToday: "", Problems: nil,
		FileStat: domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}},
	}
}
