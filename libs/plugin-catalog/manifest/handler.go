package manifest

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-catalog/when"
)

// ErrInvalidHandler is a handler or a trigger declared wrongly.
var ErrInvalidHandler = errors.New("invalid handler")

// Handler is the plugin's handler of a shared command of libs/protocol/commands.json. The code of the
// plugin registers it under the full id of the command.
type Handler struct {
	// Command is the full id of the shared command, such as app:file.createText.
	Command string
	// Title is what a person reads the handler as.
	Title string
	// When is the condition under which the handler handles the command over the target; empty is always.
	When string
}

// Trigger runs the plugin's own handler of a shared command on an event.
type Trigger struct {
	// On is an event of libs/protocol/commands.json, such as file.added.
	On string
	// Command is the full id of the shared command the plugin handles.
	Command string
	// When is the condition under which the event runs the handler, the settings as config; empty is always.
	When string
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

// listHandlerIssues checks that each handler handles a shared command once, with a title and a
// condition that can be read, and that each trigger runs on an event a handler of the plugin.
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
	for _, declared := range m.Contributes.Triggers {
		if !IsTriggerEvent(declared.On) {
			issues = append(issues, fmt.Errorf("%w: a trigger runs on %q, which is no event", ErrInvalidHandler, declared.On))
		}
		if !observed[declared.Command] {
			issues = append(issues, fmt.Errorf("%w: a trigger runs %q, which the plugin does not handle", ErrInvalidHandler, declared.Command))
		}
		if _, err := when.Parse(declared.When); err != nil {
			issues = append(issues, fmt.Errorf("%w: the trigger of %q: when: %w", ErrInvalidHandler, declared.Command, err))
		}
	}
	return issues
}
