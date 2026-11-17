package settings

import "strings"

// Editor is the settings section of how editor text is drawn: the value of each editor.* key the
// declaration of the shell holds, by the name below the section. The declaration gives each its
// type, its bounds and its default.
type Editor map[string]any

const editorKeyPrefix = "editor."

// DefaultEditor returns the editor settings the declaration of the shell gives.
func DefaultEditor() Editor {
	e := Editor{}
	for _, declared := range readAppDeclaration() {
		if strings.HasPrefix(declared.Key, editorKeyPrefix) {
			e[strings.TrimPrefix(declared.Key, editorKeyPrefix)] = declared.Default
		}
	}
	return e
}

func (e *Editor) reset(key string) {
	if !strings.HasPrefix(key, editorKeyPrefix) {
		return
	}
	if *e == nil {
		*e = Editor{}
	}
	(*e)[strings.TrimPrefix(key, editorKeyPrefix)] = getAppDefault(key)
}
