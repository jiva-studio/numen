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
		if name, isEditor := strings.CutPrefix(declared.Key, editorKeyPrefix); isEditor {
			e[name] = declared.Default
		}
	}
	return e
}

func (e *Editor) reset(key string) {
	name, isEditor := strings.CutPrefix(key, editorKeyPrefix)
	if !isEditor {
		return
	}
	if *e == nil {
		*e = Editor{}
	}
	(*e)[name] = getAppDefault(key)
}
