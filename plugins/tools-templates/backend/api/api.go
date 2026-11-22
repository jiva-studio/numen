// Package api declares the JSON messages of the templates service the plugin's backend serves to its
// own frontend. Keys are lowerCamelCase; a failure is a server.APIError, not a message field.
package api

// Service is the service name the manifest declares.
const Service = "md.numen.tools.templates"

const (
	MethodListFields = "ListFields"
	MethodCreateNote = "CreateNote"
)

// FieldKind is what a field asks for.
type FieldKind string

// The kinds of a field.
const (
	KindText   FieldKind = "text"
	KindSelect FieldKind = "select"
)

// ListFieldsRequest asks what a template asks of the person who makes a note from it.
type ListFieldsRequest struct {
	Template string `json:"template"`
	Folder   string `json:"folder"`
}

// Field is a question of a template. Options is empty for a text field.
type Field struct {
	Kind    FieldKind `json:"kind"`
	Label   string    `json:"label"`
	Options []string  `json:"options"`
	Default string    `json:"default"`
}

// ListFieldsResponse lists the fields of a template, in the order they are asked. Etag tells the template
// they were read from, and CreateNote gives it back.
type ListFieldsResponse struct {
	Fields []Field `json:"fields"`
	Etag   string  `json:"etag"`
}

// CreateNoteRequest asks for a note made from a template. Values answers the fields by label, and Etag is
// the one ListFields gave for the template.
type CreateNoteRequest struct {
	Template string            `json:"template"`
	Folder   string            `json:"folder"`
	Values   map[string]string `json:"values"`
	Etag     string            `json:"etag"`
}

// CreateNoteResponse names the note that was made. Warning is what the person should know of a note that was
// made all the same, and empty when there is nothing.
type CreateNoteResponse struct {
	Path    string `json:"path"`
	Warning string `json:"warning"`
}
