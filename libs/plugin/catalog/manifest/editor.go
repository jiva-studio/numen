package manifest

import (
	"fmt"
	"regexp"
	"strings"
)

var editorExtensionIDPattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

// Editor is what a plugin adds to the note editor.
type Editor struct {
	Extensions []EditorExtension
}

// EditorExtension is a CodeMirror extension the plugin's frontend registers in the note editor.
type EditorExtension struct {
	ID          string
	Title       string
	Description string
}

func (m Manifest) listEditorIssues() []error {
	var issues []error
	if len(m.Contributes.Editor.Extensions) > 0 && !m.HasFrontend() {
		issues = append(issues, fmt.Errorf("%w: the plugin declares editor extensions and has no frontend to register them", ErrInvalidContribution))
	}
	observed := map[string]bool{}
	for _, extension := range m.Contributes.Editor.Extensions {
		switch {
		case !editorExtensionIDPattern.MatchString(extension.ID):
			issues = append(issues, fmt.Errorf("%w: editor extension id %q is not words in camelCase", ErrInvalidContribution, extension.ID))
		case observed[extension.ID]:
			issues = append(issues, fmt.Errorf("%w: editor extension %q is declared twice", ErrInvalidContribution, extension.ID))
		}
		observed[extension.ID] = true
		if strings.TrimSpace(extension.Title) == "" {
			issues = append(issues, fmt.Errorf("%w: editor extension %q has no title", ErrInvalidContribution, extension.ID))
		}
		if strings.TrimSpace(extension.Description) == "" {
			issues = append(issues, fmt.Errorf("%w: editor extension %q has no description", ErrInvalidContribution, extension.ID))
		}
	}
	return issues
}
