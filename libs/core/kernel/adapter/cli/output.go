package cli

import (
	"fmt"
	"io"
	"sync"
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

// Printf writes a formatted string as fmt.Fprintf does.
func (r *reporter) Printf(format string, a ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err == nil {
		_, r.err = fmt.Fprintf(r.out, format, a...)
	}
}

// Println writes its operands and a newline as fmt.Fprintln does.
func (r *reporter) Println(a ...any) {
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
