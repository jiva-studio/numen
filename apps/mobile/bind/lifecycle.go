package bind

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// lifecycleTimeout is how long Suspend and Resume wait for the plugins: the grace a plugin has to send its last
// changes, and a margin.
const lifecycleTimeout = 25 * time.Second

// lifecycleCall asks one linked plugin to follow the screen. The plugin answers the methods Suspend and Resume of
// its service.
type lifecycleCall struct {
	handler http.Handler
	service string
	token   string
}

// Suspend tells the plugins that the application left the screen and waits until they have stopped their work, at
// most lifecycleTimeout. A core that is not running does nothing.
func (c *Core) Suspend() error { return c.notifyPlugins("Suspend") }

// Resume tells the plugins that the application is on the screen again. A core that is not running does nothing.
func (c *Core) Resume() error { return c.notifyPlugins("Resume") }

func (c *Core) notifyPlugins(method string) error {
	c.mu.Lock()
	running := c.running
	c.mu.Unlock()
	if running == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), lifecycleTimeout)
	defer cancel()
	var failures []error
	for _, call := range running.lifecycles {
		if err := call.send(ctx, method); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (l lifecycleCall) send(ctx context.Context, method string) error {
	url := "http://plugin/" + l.service + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+l.token)
	req.Header.Set("Content-Type", "application/json")
	answer := &answerBuffer{header: http.Header{}, status: http.StatusOK}
	l.handler.ServeHTTP(answer, req)
	if answer.status != http.StatusOK {
		return fmt.Errorf("%s of %s answered %d: %s", method, l.service, answer.status, answer.body.String())
	}
	return nil
}

// answerBuffer keeps the answer of a handler served in the process.
type answerBuffer struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (a *answerBuffer) Header() http.Header { return a.header }

func (a *answerBuffer) WriteHeader(status int) { a.status = status }

func (a *answerBuffer) Write(p []byte) (int, error) { return a.body.Write(p) }
