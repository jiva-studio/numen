package pluginport

import (
	pathpkg "path"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// GetTargetValues gives the target variables of a when condition for the path a command runs over:
// the root folder for no path, otherwise the file with its content type and, for a note, its note
// type. A path the content type table knows no type of may be a folder or a file, so whether it is a
// folder or a file is left out, and a condition that depends on it is Unknown.
func GetTargetValues(path, contentType, noteType string) map[string]any {
	if path == "" {
		return map[string]any{
			"target.isRoot": true, "target.isFolder": true, "target.isMultiple": false, "target.isNote": false,
			"target.isFile": false, "target.contentType": "", "target.noteType": "",
		}
	}
	values := map[string]any{"target.isRoot": false, "target.isMultiple": false, "target.noteType": noteType}
	isNote := strings.EqualFold(pathpkg.Ext(path), domain.NoteExtension)
	values["target.isNote"] = isNote
	if contentType == "" || contentType == string(domain.UnknownContentType) {
		values["target.contentType"] = ""
		if !isNote {
			return values
		}
	} else {
		values["target.contentType"] = contentType
	}
	values["target.isFolder"] = false
	values["target.isFile"] = !isNote
	return values
}
