package webui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// The window asks the hour a day of review begins at, and sets it from the
// settings screen.

// TestTheHourADayBeginsAtIsAnsweredByTheNextQuestion. The screen sets it, the
// file is written, and the window reads what was written.
func TestTheHourADayBeginsAtIsAnsweredByTheNextQuestion(t *testing.T) {
	f := opening(t, nil, nil, true)

	said, err := f.client.Reviewing(t.Context(), connect.NewRequest(&v1.ReviewingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if said.Msg.GetDayStarts() != "04:00" {
		t.Fatalf("an installation nobody has configured begins the day at %q",
			said.Msg.GetDayStarts())
	}

	if _, err := f.client.ChooseReviewing(t.Context(),
		connect.NewRequest(&v1.ChooseReviewingRequest{DayStarts: "05:30"})); err != nil {
		t.Fatal(err)
	}

	said, err = f.client.Reviewing(t.Context(), connect.NewRequest(&v1.ReviewingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if said.Msg.GetDayStarts() != "05:30" {
		t.Errorf("the day begins at %q", said.Msg.GetDayStarts())
	}
}

// An hour the setting does not take is not written, and the settings are left
// as they are.
func TestAnHourADayOfReviewCannotBeginAtIsNotWritten(t *testing.T) {
	f := opening(t, nil, nil, true)

	for _, one := range []string{"13:00", "24:00", "half past four", ""} {
		if _, err := f.client.ChooseReviewing(t.Context(),
			connect.NewRequest(&v1.ChooseReviewingRequest{DayStarts: one})); err == nil {
			t.Errorf("a day was made to begin at %q", one)
		}
	}

	said, err := f.client.Reviewing(t.Context(), connect.NewRequest(&v1.ReviewingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if said.Msg.GetDayStarts() != "04:00" {
		t.Errorf("the day begins at %q", said.Msg.GetDayStarts())
	}
}

// The hour a person sets is written where they will read it, and every other
// byte of the file is left as they typed it.
func TestSettingTheHourLeavesTheRestOfTheFileAlone(t *testing.T) {
	f := opening(t, nil, nil, true)
	path := filepath.Join(filepath.Dir(f.settings), "numen.json")
	if err := os.WriteFile(path, []byte("{\n  \"appearance\": {\"text_scale\": 1.5}\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := f.client.ChooseReviewing(t.Context(),
		connect.NewRequest(&v1.ChooseReviewingRequest{DayStarts: "06:00"})); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"text_scale": 1.5`) {
		t.Errorf("the file was rewritten\n%s", raw)
	}
	if !strings.Contains(string(raw), `"day_starts": "06:00"`) {
		t.Errorf("the hour was not written\n%s", raw)
	}
}
