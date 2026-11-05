package service

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

func newConfiguredBackend(t *testing.T, values map[string]string) *Backend {
	t.Helper()
	env := vaulttest.New(t, map[string]string{"Term.md": getDeck()["Term.md"]})
	b := New(Config{Vault: env.Host, Store: env.Store, Now: vaulttest.NewClock()})
	b.Configure(values)
	return b
}

func TestDayStartIsReadFromTheCamelCaseKey(t *testing.T) {
	b := newConfiguredBackend(t, map[string]string{"dayStartTime": "06:30"})

	if got, want := b.scenarios().day.Starts, 6*time.Hour+30*time.Minute; got != want {
		t.Errorf("day starts %v past midnight, want %v", got, want)
	}
}

func TestDayStartIsNotReadFromTheSnakeCaseKey(t *testing.T) {
	b := newConfiguredBackend(t, map[string]string{"day_start_time": "06:30"})

	if got := b.scenarios().day.Starts; got != review.DayStartTime {
		t.Errorf("day starts %v past midnight, want the default %v", got, review.DayStartTime)
	}
}
