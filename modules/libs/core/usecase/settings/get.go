package settings

import (
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Bounds is how far a setting holding a number goes, at each end.
type Bounds struct{ Least, Most float64 }

// Current is the settings file as JSON and where it stands, with the models and
// limits that go with the settings.
type Current struct {
	Written                  string
	Path                     string
	Models                   []Model
	NestedHeadingLimitBounds Bounds
	LatestDayStartTime       string
	Day                      string
	Plugins                  []PluginInfo
}

// Catalogue reads the models the settings can name and the plugins, with the
// models each plugin answers completions with. Both are read at once, from one
// listing of the agent programs.
type Catalogue func() ([]Model, []PluginInfo)

// Get reads every setting, the models a setting can name, and the limits the
// settings are refused outside.
type Get struct {
	read                     func() (written, path string, err error)
	catalogue                Catalogue
	nestedHeadingLimitBounds Bounds
	latestDayStartTime       string
	day                      review.Day
	now                      port.Clock
}

func NewGet(
	read func() (written, path string, err error),
	catalogue Catalogue,
	nestedHeadingLimitBounds Bounds,
	latestDayStartTime string,
	day review.Day,
	now port.Clock,
) *Get {
	return &Get{
		read:                     read,
		catalogue:                catalogue,
		nestedHeadingLimitBounds: nestedHeadingLimitBounds,
		latestDayStartTime:       latestDayStartTime,
		day:                      day,
		now:                      now,
	}
}

func (u *Get) Execute() (Current, error) {
	written, path, err := u.read()
	if err != nil {
		return Current{}, err
	}
	var models []Model
	var pluginInfos []PluginInfo
	if u.catalogue != nil {
		models, pluginInfos = u.catalogue()
	}
	return Current{
		Written:                  written,
		Path:                     path,
		Models:                   models,
		NestedHeadingLimitBounds: u.nestedHeadingLimitBounds,
		LatestDayStartTime:       u.latestDayStartTime,
		Day:                      u.day.GetName(u.now()),
		Plugins:                  pluginInfos,
	}, nil
}
