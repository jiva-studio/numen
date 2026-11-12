// Package api declares the JSON messages of the index stats service the plugin's backend serves to its
// own frontend. Keys are lowerCamelCase; a failure is a server.APIError, not a message field.
package api

// Service is the service name the manifest declares.
const Service = "md.numen.tools.index.stats"

const MethodGetStats = "GetStats"

// Kind is the sort of source the file type of a source tells.
type Kind string

const (
	// KindNotes is the notes of the vault.
	KindNotes Kind = "notes"
	// KindPDF is the PDF documents.
	KindPDF Kind = "pdf"
	// KindEPUB is the EPUB books.
	KindEPUB Kind = "epub"
	// KindMedia is the audio and video files, which the index holds as transcripts.
	KindMedia Kind = "media"
	// KindOther is every other file.
	KindOther Kind = "other"
)

// GetStatsRequest asks for what the index holds now.
type GetStatsRequest struct{}

// KindCount is the sources of one kind and the chunks that can carry a vector among those they hold.
type KindCount struct {
	Kind    Kind  `json:"kind"`
	Sources int64 `json:"sources"`
	Chunks  int64 `json:"chunks"`
}

// Stats is what the index holds, and the model, profile and speed that fill it.
type Stats struct {
	Vectors       int64       `json:"vectors"`
	Chunks        int64       `json:"chunks"`
	Waiting       int64       `json:"waiting"`
	Sources       int64       `json:"sources"`
	Kinds         []KindCount `json:"kinds"`
	Model         string      `json:"model"`
	Recipe        string      `json:"recipe"`
	Dimensions    int64       `json:"dimensions"`
	Profile       string      `json:"profile"`
	Threads       int         `json:"threads"`
	Speed         *float64    `json:"speed,omitempty"`
	ChangedAt     string      `json:"changedAt,omitempty"`
	DatabaseBytes int64       `json:"databaseBytes"`
	VectorBytes   int64       `json:"vectorBytes"`
}
