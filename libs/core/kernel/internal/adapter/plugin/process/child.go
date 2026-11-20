package process

import (
	"os"
	"sync"
)

// Child holds the external process attached to one supervisor and its scheduling state.
type Child struct {
	mu      sync.Mutex
	process *os.Process
	nice    *int
}

// SetProcess attaches the child or clears it when its run ends.
func (c *Child) SetProcess(child *os.Process) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.process != child {
		c.process = child
		c.nice = nil
	}
}

func (c *Child) GetProcess() *os.Process {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.process
}

func (c *Child) GetProcessID() int {
	if child := c.GetProcess(); child != nil {
		return child.Pid
	}
	return 0
}

// SetNice attempts each profile priority once per attached process.
func (c *Child) SetNice(nice int) error {
	return c.applyNice(nice, SetNice)
}

func (c *Child) applyNice(nice int, set func(int, int) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.process == nil || (c.nice != nil && *c.nice == nice) {
		return nil
	}
	c.nice = &nice
	return set(c.process.Pid, nice)
}
