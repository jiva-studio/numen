// Package api declares the JSON messages of the updater service the plugin's backend serves to its
// own frontend. Keys are lowerCamelCase; a failure is a server.APIError, not a message field.
package api

// Service is the service name the manifest declares.
const Service = "md.numen.tools.updater"

// Method names of the service.
const (
	MethodCheck     = "Check"
	MethodGetStatus = "GetStatus"
)

// State is what the last check found.
type State string

const (
	// StateUnchecked is the state before the first check.
	StateUnchecked State = "unchecked"
	// StateCurrent means the running build is the newest the endpoint publishes.
	StateCurrent State = "current"
	// StateAvailable means the endpoint publishes a newer build; Status.Update describes it.
	StateAvailable State = "available"
	// StateUnreachable means the endpoint did not answer with a manifest. It is not an error of the
	// call: a machine without a network is in this state.
	StateUnreachable State = "unreachable"
)

// Platform is the operating system of a build.
type Platform string

const (
	PlatformDarwin  Platform = "darwin"
	PlatformWindows Platform = "windows"
	PlatformLinux   Platform = "linux"
)

// App is the running build the frontend reports.
type App struct {
	Version  string   `json:"version"`
	Build    int64    `json:"build"`
	Platform Platform `json:"platform"`
}

// CheckRequest asks the endpoint for the newest build and compares it with the running one.
type CheckRequest struct {
	App App `json:"app"`
}

// GetStatusRequest asks for the result of the last check without a request to the endpoint.
type GetStatusRequest struct{}

// File is the file of a release for the running platform. The checksum and the size are what a
// download of the file is verified against.
type File struct {
	Name   string `json:"name"`
	Arch   string `json:"arch"`
	Kind   string `json:"kind"`
	URL    string `json:"url"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// Update describes a build newer than the running one.
type Update struct {
	Version     string `json:"version"`
	Build       int64  `json:"build"`
	Channel     string `json:"channel"`
	ReleasedAt  string `json:"releasedAt"`
	NotesURL    string `json:"notesUrl"`
	DownloadURL string `json:"downloadUrl"`
	File        *File  `json:"file,omitempty"`
}

// Status is the result of the last check.
type Status struct {
	State  State   `json:"state"`
	Update *Update `json:"update,omitempty"`
}
