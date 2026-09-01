package settings_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
)

// The hour a day of review begins at is read from the file.
func TestTheHourADayBeginsAt(t *testing.T) {
	cfg, err := settings.At(write(t, `{"review": {"day_starts": "03:30"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DayStarts() != 3*time.Hour+30*time.Minute {
		t.Errorf("the day begins %v past midnight", cfg.DayStarts())
	}
	if len(cfg.Said) != 0 {
		t.Errorf("said = %v", cfg.Said)
	}
}

// An installation nobody has configured begins the day where this application
// begins it.
func TestAFileNamingNoHour(t *testing.T) {
	cfg, err := settings.At(write(t, `{"appearance": {"text_scale": 1}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DayStarts() != settings.DefaultStarts() {
		t.Errorf("the day begins %v past midnight", cfg.DayStarts())
	}
}

// Something that is not an hour of the day is one line of a band, and the
// default stands. The file is left as the person wrote it.
func TestAnHourTheDayDoesNotBeginAt(t *testing.T) {
	for _, written := range []string{
		`{"review": {"day_starts": "half past four"}}`,
		`{"review": {"day_starts": "18:00"}}`,
		`{"review": {"day_starts": "4"}}`,
	} {
		cfg, err := settings.At(write(t, written))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.DayStarts() != settings.DefaultStarts() {
			t.Errorf("%s: the day begins %v past midnight", written, cfg.DayStarts())
		}
		if len(cfg.Said) != 1 || !strings.Contains(cfg.Said[0], "review.day_starts") {
			t.Errorf("%s: said = %v", written, cfg.Said)
		}
	}
}
