package manifest

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
)

var (
	// ErrInvalidHandler is an entry of contributes.handlers declared wrongly.
	ErrInvalidHandler = errors.New("invalid handler")
	// ErrInvalidTrigger is an entry of contributes.triggers declared wrongly.
	ErrInvalidTrigger = errors.New("invalid trigger")
)

// Handler is the plugin's handler of a shared command of libs/core/protocol/commands.json. The code of the
// plugin registers it under the full id of the command.
type Handler struct {
	Command string
	Title   string
	When    string
}

// Trigger runs the plugin's own handler of a shared command over the file of an event.
type Trigger struct {
	Event   string
	Command string
	When    string
}

// GetHandler returns the plugin's handler of the shared command, and false where it has none.
func (m Manifest) GetHandler(command string) (Handler, bool) {
	for _, declared := range m.Contributes.Handlers {
		if declared.Command == command {
			return declared, true
		}
	}
	return Handler{}, false
}

func (m Manifest) listHandlerIssues() []error {
	var issues []error
	observed := map[string]bool{}
	for _, declared := range m.Contributes.Handlers {
		switch {
		case !IsSharedCommand(declared.Command):
			issues = append(issues, fmt.Errorf("%w: %q is no shared command", ErrInvalidHandler, declared.Command))
		case observed[declared.Command]:
			issues = append(issues, fmt.Errorf("%w: %q is handled twice", ErrInvalidHandler, declared.Command))
		}
		observed[declared.Command] = true
		if strings.TrimSpace(declared.Title) == "" {
			issues = append(issues, fmt.Errorf("%w: the handler of %q has no title", ErrInvalidHandler, declared.Command))
		}
		if _, err := when.Parse(declared.When); err != nil {
			issues = append(issues, fmt.Errorf("%w: the handler of %q: when: %w", ErrInvalidHandler, declared.Command, err))
		}
	}
	if len(m.Contributes.Handlers) > 0 && !m.HasBackend() && !m.HasFrontend() {
		issues = append(issues, fmt.Errorf("%w: the plugin declares handlers and has no frontend or backend to run them", ErrInvalidHandler))
	}
	return issues
}

func (m Manifest) listTriggerIssues() []error {
	var issues []error
	for _, declared := range m.Contributes.Triggers {
		if !IsEvent(declared.Event) {
			issues = append(issues, fmt.Errorf("%w: %q is no event", ErrInvalidTrigger, declared.Event))
		}
		if _, isHandled := m.GetHandler(declared.Command); !isHandled {
			issues = append(issues, fmt.Errorf("%w: on %q runs %q, which the plugin does not handle", ErrInvalidTrigger, declared.Event, declared.Command))
		}
		if _, err := when.Parse(declared.When); err != nil {
			issues = append(issues, fmt.Errorf("%w: on %q: when: %w", ErrInvalidTrigger, declared.Event, err))
		}
	}
	return issues
}
