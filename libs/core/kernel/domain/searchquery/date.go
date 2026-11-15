package searchquery

import (
	"errors"
	"regexp"
	"time"
)

// ErrInvalidDate is a text that date() and a date comparison of a property do not read.
var ErrInvalidDate = errors.New("a date is written as YYYY-MM-DD or YYYY-MM-DDTHH:MM[:SS] with an optional Z or ±HH:MM")

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}(:\d{2}(\.\d{1,9})?)?(Z|[+-]\d{2}:\d{2})?)?$`)

// ParseDate reads YYYY-MM-DD, YYYY-MM-DDTHH:MM or YYYY-MM-DDTHH:MM:SS with up to nine digits of a fraction,
// each time form with an optional Z or ±HH:MM, and a space in place of the T. A text without an offset is
// read in zone; a date alone is midnight there.
func ParseDate(text string, zone *time.Location) (time.Time, error) {
	if len(text) > len(time.DateOnly) && text[len(time.DateOnly)] == ' ' {
		text = text[:len(time.DateOnly)] + "T" + text[len(time.DateOnly)+1:]
	}
	match := datePattern.FindStringSubmatch(text)
	if match == nil {
		return time.Time{}, ErrInvalidDate
	}
	layout := time.DateOnly
	switch {
	case match[2] != "":
		layout = "2006-01-02T15:04:05"
	case match[1] != "":
		layout = "2006-01-02T15:04"
	}
	var parsed time.Time
	var err error
	if match[4] != "" {
		parsed, err = time.Parse(layout+"Z07:00", text)
	} else {
		parsed, err = time.ParseInLocation(layout, text, zone)
	}
	if err != nil {
		return time.Time{}, ErrInvalidDate
	}
	return parsed, nil
}
