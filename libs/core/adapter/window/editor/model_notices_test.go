package editor

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
)

func TestAConnectionSetAsideIsAKindOfNoticeInTheListOfTasks(t *testing.T) {
	tasks := run.NewTasks()
	notices := newModelNotices(tasks)

	notices.ReportSetAside("c0000001", "OpenRouter", connectionport.FailureReasonPaymentRequired)

	list := tasks.List()
	if len(list) != 1 || list[0].Kind != run.KindNotice || list[0].Error != "" {
		t.Fatalf("tasks = %+v, want one notice that is not a failure", list)
	}
	if !strings.Contains(list[0].About, "OpenRouter") || !strings.Contains(list[0].About, "payment required") {
		t.Errorf("About = %q, want the connection and the reason", list[0].About)
	}
}

func TestEachConnectionAndReasonIsOneTask(t *testing.T) {
	tasks := run.NewTasks()
	notices := newModelNotices(tasks)

	notices.ReportSetAside("c0000001", "OpenRouter", connectionport.FailureReasonPaymentRequired)
	notices.ReportSetAside("c0000001", "OpenRouter", connectionport.FailureReasonPaymentRequired)
	notices.ReportSetAside("c0000001", "OpenRouter", connectionport.FailureReasonRateLimited)
	notices.ReportSetAside("c0000002", "Ollama", connectionport.FailureReasonPaymentRequired)

	if got := len(tasks.List()); got != 3 {
		t.Errorf("tasks = %+v, want one for each connection and reason", tasks.List())
	}
}
