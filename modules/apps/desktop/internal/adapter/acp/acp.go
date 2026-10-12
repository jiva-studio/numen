// Package acp reaches external coding agents communicating over JSON streams.
package acp

import (
	"context"
	"os/exec"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Endpoint is where the agent reaches this vault's MCP tools.
type Endpoint struct {
	URL   string
	Token string
}

// Agent reaches an external agent CLI using standard Agent Client Protocol (ACP).
type Agent struct {
	// Program names the agent identifier ("antigravity", "codex", etc.)
	Program string
	// Command starts it. Empty uses discovered binary from path or known places.
	Command []string
	// Root is the folder the agent is started in.
	Root string
	// Tools is where it reaches this vault over MCP.
	Tools Endpoint
	// Model is which model answers.
	Model string
	// Turns is max step count.
	Turns int
	// ErrorHandler is notified of background execution errors.
	ErrorHandler func(error)

	carried struct {
		sync.Mutex
		sessions map[string]string
	}

	taken struct {
		sync.Mutex
		running map[*work]bool
		isShut  bool
	}
}

type work struct {
	cmd          *exec.Cmd
	stop         context.CancelFunc
	conversation string
	steps        chan port.Step
	reader       sync.WaitGroup
	once         sync.Once
	cleanup      func()
}

func (w *work) GetSteps() <-chan port.Step { return w.steps }

func (w *work) Stop() error {
	w.once.Do(w.stop)
	w.reader.Wait()
	return nil
}

func (a *Agent) registerWork(w *work) bool {
	a.taken.Lock()
	defer a.taken.Unlock()

	if a.taken.isShut {
		return false
	}
	if a.taken.running == nil {
		a.taken.running = map[*work]bool{}
	}
	a.taken.running[w] = true
	return true
}

func (a *Agent) unregisterWork(w *work) {
	a.taken.Lock()
	defer a.taken.Unlock()
	delete(a.taken.running, w)
}

// Close stops every running agent process.
func (a *Agent) Close() error {
	a.taken.Lock()
	a.taken.isShut = true
	running := make([]*work, 0, len(a.taken.running))
	for w := range a.taken.running {
		running = append(running, w)
	}
	a.taken.running = nil
	a.taken.Unlock()

	for _, w := range running {
		w.once.Do(w.stop)
	}
	for _, w := range running {
		w.reader.Wait()
	}
	return nil
}

// EndConversation ends a conversation session.
func (a *Agent) EndConversation(_ context.Context, conversation string) error {
	if conversation == "" {
		return nil
	}

	a.taken.Lock()
	answering := make([]*work, 0, len(a.taken.running))
	for w := range a.taken.running {
		if w.conversation == conversation {
			answering = append(answering, w)
		}
	}
	a.taken.Unlock()

	for _, w := range answering {
		w.once.Do(w.stop)
	}
	for _, w := range answering {
		w.reader.Wait()
	}

	a.carried.Lock()
	defer a.carried.Unlock()
	delete(a.carried.sessions, conversation)
	return nil
}

func (a *Agent) resolveCommand() (string, []string) {
	if len(a.Command) > 0 {
		return a.Command[0], a.Command[1:]
	}
	if a.Program == "codex" {
		return FindCodex(), nil
	}
	return FindAntigravity(), nil
}
