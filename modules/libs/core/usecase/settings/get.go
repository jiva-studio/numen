package settings

import (
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Get reads every setting, the models a setting can name, and the limits the
// settings are refused outside.
type Get struct {
	read                     func() (written, path string, err error)
	models                   func() []port.Model
	nestedHeadingLimitBounds Bounds
	latestDayStarts          string
	day                      review.Day
	now                      port.Clock
}

// Settings is every setting as JSON and the file it stands in, with the models
// and limits that go with them.
type Settings struct {
	Written                  string
	Path                     string
	Models                   []port.Model
	NestedHeadingLimitBounds Bounds
	LatestDayStarts          string
	Day                      string
}

func NewGet(
	read func() (written, path string, err error),
	models func() []port.Model,
	nestedHeadingLimitBounds Bounds,
	latestDayStarts string,
	day review.Day,
	now port.Clock,
) Get {
	return Get{
		read:                     read,
		models:                   models,
		nestedHeadingLimitBounds: nestedHeadingLimitBounds,
		latestDayStarts:          latestDayStarts,
		day:                      day,
		now:                      now,
	}
}

func (u Get) Execute() (Settings, error) {
	written, path, err := u.read()
	if err != nil {
		return Settings{}, err
	}
	var models []port.Model
	if u.models != nil {
		models = u.models()
	}
	return Settings{
		Written:                  written,
		Path:                     path,
		Models:                   models,
		NestedHeadingLimitBounds: u.nestedHeadingLimitBounds,
		LatestDayStarts:          u.latestDayStarts,
		Day:                      u.day.GetName(u.now()),
	}, nil
}
