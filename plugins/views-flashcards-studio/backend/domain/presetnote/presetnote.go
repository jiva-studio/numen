// Package presetnote is a preset as its note holds it: what a read of the note found, the link
// type a deck names its preset under, and the frontmatter keys of the settings a curve is steered
// by.
package presetnote

import (
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
)

// LinkType is what a deck's link to its preset carries under `type`.
const LinkType = "preset"

// The frontmatter keys of the two settings a goal is steered by.
const (
	MinutesADayKey = "minutes_a_day"
	RetentionKey   = "retention"
)

// ErrOutOfBounds is a setting outside what it may be. Nothing is written, and
// the settings are weighed before the file is opened.
var ErrOutOfBounds = errors.New("this setting is outside what a preset may hold")

// Contents is one preset as returned by a read.
type Contents struct {
	Path       string
	Outcome    edit.ReadOutcome
	Type       domain.NoteType
	Settings   review.Preset
	Stops      review.StopReason
	StopsToday review.StopReason
	Problems   []string
	FileStat   domain.FileStat
}

// CreateDefault returns the preset of a deck scheduled by no preset.
func CreateDefault() Contents {
	return Contents{
		Path: "", Outcome: edit.Ok, Type: "", Settings: review.Defaults(), Stops: "", StopsToday: "", Problems: nil,
		FileStat: domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}},
	}
}
