package settings_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
)

// The hour a day of review begins at is read from the file.
func TestTheHourADayBeginsAt(t *testing.T) {
	cfg, err := openAt(write(t, `{"review": {"day_start_time": "03:30"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GetDayStartTime() != 3*time.Hour+30*time.Minute {
		t.Errorf("the day begins %v past midnight", cfg.GetDayStartTime())
	}
	if len(cfg.Notices) != 0 {
		t.Errorf("said = %v", cfg.Notices)
	}
}

// An installation nobody has configured begins the day where this application
// begins it.
func TestAFileNamingNoHour(t *testing.T) {
	cfg, err := openAt(write(t, `{"appearance": {"text_scale": 1}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GetDayStartTime() != settings.DefaultDayStart() {
		t.Errorf("the day begins %v past midnight", cfg.GetDayStartTime())
	}
}

// Something that is not an hour of the day is one line of a band, and the
// default stands. The file is left as the person wrote it.
func TestAnHourTheDayDoesNotBeginAt(t *testing.T) {
	for _, written := range []string{
		`{"review": {"day_start_time": "half past four"}}`,
		`{"review": {"day_start_time": "18:00"}}`,
		`{"review": {"day_start_time": "4"}}`,
	} {
		cfg, err := openAt(write(t, written))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.GetDayStartTime() != settings.DefaultDayStart() {
			t.Errorf("%s: the day begins %v past midnight", written, cfg.GetDayStartTime())
		}
		if len(cfg.Notices) != 1 || !strings.Contains(cfg.Notices[0], "review.day_start_time") {
			t.Errorf("%s: said = %v", written, cfg.Notices)
		}
	}
}

// A file written under the old name is read as the hour it names, and the field
// takes the new name where it stands.
func TestAnHourWrittenUnderTheOldName(t *testing.T) {
	path := write(t, `{"review": {"day_starts": "03:30"}}`)
	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GetDayStartTime() != 3*time.Hour+30*time.Minute {
		t.Errorf("the day begins %v past midnight", cfg.GetDayStartTime())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"day_start_time": "03:30"`) || strings.Contains(string(raw), "day_starts") {
		t.Errorf("the file is now %s", raw)
	}
}

// A file that names the hour under both names keeps both, and the new one is
// the hour.
func TestAnHourWrittenUnderBothNames(t *testing.T) {
	path := write(t, `{"review": {"day_starts": "03:30", "day_start_time": "05:00"}}`)
	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GetDayStartTime() != 5*time.Hour {
		t.Errorf("the day begins %v past midnight", cfg.GetDayStartTime())
	}
}
