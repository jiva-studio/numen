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

// Get reads every setting, the models a setting can name, and the limits the
// settings are refused outside.
type Get struct {
	read                     func() (written, path string, err error)
	models                   func() []Model
	nestedHeadingLimitBounds Bounds
	latestDayStartTime       string
	day                      review.Day
	now                      port.Clock
	plugins                  func() []PluginInfo
}

func NewGet(
	read func() (written, path string, err error),
	models func() []Model,
	nestedHeadingLimitBounds Bounds,
	latestDayStartTime string,
	day review.Day,
	now port.Clock,
	plugins func() []PluginInfo,
) *Get {
	return &Get{
		read:                     read,
		models:                   models,
		nestedHeadingLimitBounds: nestedHeadingLimitBounds,
		latestDayStartTime:       latestDayStartTime,
		day:                      day,
		now:                      now,
		plugins:                  plugins,
	}
}

func (u *Get) Execute() (Current, error) {
	written, path, err := u.read()
	if err != nil {
		return Current{}, err
	}
	var models []Model
	if u.models != nil {
		models = u.models()
	}
	var pluginInfos []PluginInfo
	if u.plugins != nil {
		pluginInfos = u.plugins()
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
