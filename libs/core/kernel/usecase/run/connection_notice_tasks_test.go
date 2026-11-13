package run_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
)

func setAside(id, name string, reason connection.FailureReason) connection.SetAside {
	return connection.SetAside{ConnectionID: id, ConnectionName: name, Reason: reason, NextName: "Ollama"}
}

func TestAConnectionSetAsideIsAKindOfNoticeInTheListOfTasks(t *testing.T) {
	tasks := run.NewTasks()
	notices := run.NewConnectionNoticeTasks(tasks)

	notices.Execute(setAside("c0000001", "OpenRouter", connection.FailureReasonPaymentRequired))

	list := tasks.List()
	if len(list) != 1 || list[0].Kind != work.KindNotice || list[0].Error != "" {
		t.Fatalf("tasks = %+v, want one notice that is not a failure", list)
	}
	if list[0].Title != "Another connection answered" {
		t.Errorf("Title = %q", list[0].Title)
	}
}

func TestTheNoticeNamesBothConnectionsInOneSentencePerReason(t *testing.T) {
	for reason, want := range map[connection.FailureReason]string{
		connection.FailureReasonUnavailable:     "OpenRouter is not available, so Ollama answered.",
		connection.FailureReasonRejectedKey:     "OpenRouter rejected the key, so Ollama answered.",
		connection.FailureReasonPaymentRequired: "OpenRouter needs payment, so Ollama answered.",
		connection.FailureReasonRateLimited:     "OpenRouter hit its rate limit, so Ollama answered.",
		connection.FailureReasonInvalidRequest:  "OpenRouter refused the request, so Ollama answered.",
	} {
		tasks := run.NewTasks()

		run.NewConnectionNoticeTasks(tasks).Execute(setAside("c0000001", "OpenRouter", reason))

		if got := tasks.List()[0].About; got != want {
			t.Errorf("%v: About = %q, want %q", reason, got, want)
		}
	}
}

func TestEachConnectionAndReasonIsOneTask(t *testing.T) {
	tasks := run.NewTasks()
	notices := run.NewConnectionNoticeTasks(tasks)

	notices.Execute(setAside("c0000001", "OpenRouter", connection.FailureReasonPaymentRequired))
	notices.Execute(setAside("c0000001", "OpenRouter", connection.FailureReasonPaymentRequired))
	notices.Execute(setAside("c0000001", "OpenRouter", connection.FailureReasonRateLimited))
	notices.Execute(setAside("c0000002", "Groq", connection.FailureReasonPaymentRequired))

	if got := len(tasks.List()); got != 3 {
		t.Errorf("tasks = %+v, want one for each connection and reason", tasks.List())
	}
}
