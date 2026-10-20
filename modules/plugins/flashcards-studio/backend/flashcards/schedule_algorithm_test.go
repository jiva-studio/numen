package flashcards_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/review"
)

type mockScheduler struct {
	name string
}

func (m mockScheduler) GetName() string { return m.name }
func (m mockScheduler) Next(s review.Schedule, at time.Time, r review.Rating) review.Schedule {
	s.Last = at
	s.Due = at.Add(24 * time.Hour)
	s.Reps++
	return s
}
func (m mockScheduler) Endings(s review.Schedule, at time.Time) (good, again review.Schedule) {
	good = m.Next(s, at, review.Good)
	again = m.Next(s, at, review.Again)
	return good, again
}
func (m mockScheduler) IsSpaced(_ review.Schedule) bool { return true }

type mockAlgorithm struct {
	calledWith []float64
}

func (m *mockAlgorithm) GetName() string { return "mock-algo" }
func (m *mockAlgorithm) CreateScheduler(retention float64) review.Scheduler {
	m.calledWith = append(m.calledWith, retention)
	return mockScheduler{name: "mock-algo"}
}

func TestScheduleAlgorithmPluggable(t *testing.T) {
	algo := &mockAlgorithm{}
	scenarios := flashcards.NewScenarios(flashcards.Config{
		Scheduler: algo,
	})

	if len(algo.calledWith) == 0 {
		t.Fatal("expected scheduler algorithm CreateScheduler to be called")
	}
	_ = scenarios
}
