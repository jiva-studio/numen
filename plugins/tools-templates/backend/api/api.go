// Package api declares the JSON messages of the templates service the plugin's backend serves to its
// own frontend. Keys are lowerCamelCase; a failure is a server.APIError, not a message field.
package api

// Service is the service name the manifest declares.
const Service = "md.numen.tools.templates"

const MethodCreateNote = "CreateNote"

// CreateNoteRequest asks for a note made from a template.
type CreateNoteRequest struct {
	Template string `json:"template"`
	Folder   string `json:"folder"`
}

// CreateNoteResponse names the note that was made.
type CreateNoteResponse struct {
	Path string `json:"path"`
}
