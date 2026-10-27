// Package contenttypeport declares what the core needs to know the content type of a file.
package contenttypeport

import "github.com/jiva-studio/numen/libs/notes/domain"

// ContentTypeTable gives a file its content type by its name.
type ContentTypeTable interface {
	// GetContentType is the type of a file by its name, and domain.UnknownContentType where the
	// table names none.
	GetContentType(name string) domain.ContentType
	// ListExtensions lists, lowercase and without the dot, the extensions that give a file a type
	// the mask covers.
	ListExtensions(mask domain.ContentType) []string
}
