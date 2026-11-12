package file

import (
	pathpkg "path"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
)

// GetMediaType finds the content type a file of the vault is served and played as, by its name.
type GetMediaType struct {
	types contenttypeport.ContentTypeTable
}

func NewGetMediaType(types contenttypeport.ContentTypeTable) *GetMediaType {
	return &GetMediaType{types: types}
}

// Execute returns the content type of the file at path, and empty where the table names none.
func (u *GetMediaType) Execute(path string) string {
	named := u.types.GetContentType(pathpkg.Base(path))
	if named == domain.UnknownContentType {
		return ""
	}
	return string(named)
}
