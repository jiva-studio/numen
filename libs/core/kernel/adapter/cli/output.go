package cli

import (
	"cmp"
	"fmt"
	"io"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
)

// reporter writes to a stream and keeps the first write error. Once a write
// has failed, later writes are skipped. It is safe for concurrent use.
type reporter struct {
	mu  sync.Mutex
	out io.Writer
	err error
}

func newReporter(out io.Writer) *reporter {
	return &reporter{out: out}
}

// Print writes its operands as fmt.Fprint does.
func (r *reporter) Print(a ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err == nil {
		_, r.err = fmt.Fprint(r.out, a...)
	}
}

// WriteFormat writes a formatted string as fmt.Fprintf does.
func (r *reporter) WriteFormat(format string, a ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err == nil {
		_, r.err = fmt.Fprintf(r.out, format, a...)
	}
}

// WriteLine writes its operands and a newline as fmt.Fprintln does.
func (r *reporter) WriteLine(a ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err == nil {
		_, r.err = fmt.Fprintln(r.out, a...)
	}
}

// Err returns the first write error, or nil when every write succeeded.
func (r *reporter) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

// FormatTaskFailures lists each failed task with its owner and title for the closing report.
func FormatTaskFailures(tasks []work.Task) []string {
	var failed []string
	for _, task := range tasks {
		if task.Error != "" {
			owner := cmp.Or(task.PluginName, task.PluginID)
			failed = append(failed, fmt.Sprintf("%s: %s: %s", owner, task.Title, task.Error))
		}
	}
	return failed
}
