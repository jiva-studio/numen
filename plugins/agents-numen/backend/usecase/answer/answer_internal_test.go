package answer

import (
	"context"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock/clocktest"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
)

func TestRunningCancelsRemainingTasksOnClose(t *testing.T) {
	r := &runningTasks{cancels: map[*context.CancelFunc]bool{}}

	var firstCanceled, secondCanceled, thirdCanceled bool
	var first context.CancelFunc = func() { firstCanceled = true }
	var second context.CancelFunc = func() { secondCanceled = true }
	var third context.CancelFunc = func() { thirdCanceled = true }

	if !r.add(&first) || !r.add(&second) || !r.add(&third) {
		t.Fatal("add rejected a task before close")
	}
	r.remove(&second)
	if len(r.cancels) != 2 {
		t.Fatalf("len(cancels) = %d, want 2", len(r.cancels))
	}

	r.close()

	if !firstCanceled || secondCanceled || !thirdCanceled {
		t.Errorf("canceled = (%v, %v, %v), want (true, false, true)", firstCanceled, secondCanceled, thirdCanceled)
	}
	if len(r.cancels) != 0 {
		t.Errorf("len(cancels) after close = %d, want 0", len(r.cancels))
	}
}

func TestHistoryIgnoresEmptyIDAndExpiresIdleOnAppend(t *testing.T) {
	fakeClock := new(clocktest.Fake)
	fakeClock.Set(time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC))
	h := newHistory(fakeClock)

	h.append("", []conversation.Message{{Role: "user", Text: "ephemeral"}})
	if len(h.conversations) != 0 {
		t.Fatalf("len(conversations) = %d, want 0", len(h.conversations))
	}

	h.append("c1", []conversation.Message{{Role: "user", Text: "first"}})
	fakeClock.Advance(20 * time.Minute)
	h.append("c1", []conversation.Message{{Role: "assistant", Text: "second"}})
	fakeClock.Advance(20 * time.Minute)
	if got := h.get("c1"); len(got) != 2 {
		t.Fatalf("len(get(c1)) = %d, want 2", len(got))
	}

	fakeClock.Advance(10 * time.Minute)
	h.append("c2", []conversation.Message{{Role: "user", Text: "fresh"}})
	if _, isKept := h.conversations["c1"]; isKept {
		t.Error("expected idle c1 to be forgotten during append of c2")
	}
}
