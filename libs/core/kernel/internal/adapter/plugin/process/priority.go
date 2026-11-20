package process

import "os"

// Priority remembers the last scheduling request for a child process.
type Priority struct {
	child *os.Process
	nice  int
}

// Apply updates a child when its identity or requested scheduling priority changes.
func (p *Priority) Apply(child *os.Process, nice int, set func(int, int) error) error {
	if child == nil || (p.child == child && p.nice == nice) {
		return nil
	}
	p.child, p.nice = child, nice
	return set(child.Pid, nice)
}
