package searchquery

import (
	"path"
	"strings"
)

// GetTitle returns the title of a document: the stored note title, else the file name without its last
// extension.
func GetTitle(filePath, noteTitle string) string {
	if noteTitle != "" {
		return noteTitle
	}
	name := path.Base(filePath)
	if dot := strings.LastIndexByte(name, '.'); dot > 0 {
		return name[:dot]
	}
	return name
}
