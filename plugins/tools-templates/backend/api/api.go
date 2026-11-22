// Package api declares the JSON messages of the templates service the plugin's backend serves to its
// own frontend. Keys are lowerCamelCase; a failure is a server.APIError, not a message field.
package api

// Service is the service name the manifest declares.
const Service = "md.numen.tools.templates"

const (
	MethodListFields = "ListFields"
	MethodCreateNote = "CreateNote"
)

// The kinds of a field.
const (
	KindText   = "text"
	KindSelect = "select"
)

// ListFieldsRequest asks what a template asks of the person who makes a note from it.
type ListFieldsRequest struct {
	Template string `json:"template"`
	Folder   string `json:"folder"`
}

// Field is a question of a template. Options is empty for a text field.
type Field struct {
	Kind    string   `json:"kind"`
	Label   string   `json:"label"`
	Options []string `json:"options"`
	Default string   `json:"default"`
}

// ListFieldsResponse lists the fields of a template, in the order they are asked.
type ListFieldsResponse struct {
	Fields []Field `json:"fields"`
}

// CreateNoteRequest asks for a note made from a template. Values answers the fields by label.
type CreateNoteRequest struct {
	Template string            `json:"template"`
	Folder   string            `json:"folder"`
	Values   map[string]string `json:"values"`
}

// CreateNoteResponse names the note that was made.
type CreateNoteResponse struct {
	Path string `json:"path"`
}
