package run_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
)

func TestAConnectionSetAsideIsAKindOfNoticeInTheListOfTasks(t *testing.T) {
	tasks := run.NewTasks()
	notices := run.NewConnectionNoticeTasks(tasks)

	notices.Execute("c0000001", "OpenRouter", connection.FailureReasonPaymentRequired)

	list := tasks.List()
	if len(list) != 1 || list[0].Kind != work.KindNotice || list[0].Error != "" {
		t.Fatalf("tasks = %+v, want one notice that is not a failure", list)
	}
	if !strings.Contains(list[0].About, "OpenRouter") || !strings.Contains(list[0].About, "payment required") {
		t.Errorf("About = %q, want the connection and the reason", list[0].About)
	}
}

func TestEachConnectionAndReasonIsOneTask(t *testing.T) {
	tasks := run.NewTasks()
	notices := run.NewConnectionNoticeTasks(tasks)

	notices.Execute("c0000001", "OpenRouter", connection.FailureReasonPaymentRequired)
	notices.Execute("c0000001", "OpenRouter", connection.FailureReasonPaymentRequired)
	notices.Execute("c0000001", "OpenRouter", connection.FailureReasonRateLimited)
	notices.Execute("c0000002", "Ollama", connection.FailureReasonPaymentRequired)

	if got := len(tasks.List()); got != 3 {
		t.Errorf("tasks = %+v, want one for each connection and reason", tasks.List())
	}
}
