// Package api declares the JSON messages of the index stats service the plugin's backend serves to its
// own frontend. Keys are lowerCamelCase; a failure is a server.APIError, not a message field.
package api

// Service is the service name the manifest declares.
const Service = "md.numen.tools.index.stats"

// MethodGetStats is the one method of the service.
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
	// Vectors is how many chunks carry a vector, and Chunks how many can. Waiting is how many do not
	// yet, and is zero once the index is embedded.
	Vectors int64 `json:"vectors"`
	Chunks  int64 `json:"chunks"`
	Waiting int64 `json:"waiting"`
	// Sources is the notes and files in the index, and Kinds splits them, a kind with none left out.
	Sources int64       `json:"sources"`
	Kinds   []KindCount `json:"kinds"`
	// Model, Recipe and Dimensions identify the embedding model in use.
	Model      string `json:"model"`
	Recipe     string `json:"recipe"`
	Dimensions int64  `json:"dimensions"`
	// Profile is the performance profile in force, and Threads how many threads it grants a plugin.
	Profile string `json:"profile"`
	Threads int    `json:"threads"`
	// Speed is the chunks embedded each second over the last ten seconds. It is present while chunks
	// wait and the plugin has watched them for a second or more.
	Speed *float64 `json:"speed,omitempty"`
	// ChangedAt is the time, in RFC 3339, of the last change of the counts the plugin saw since it
	// started. It is absent before the first change.
	ChangedAt string `json:"changedAt,omitempty"`
	// DatabaseBytes is the size of the index database on disk, and VectorBytes the size of the tables
	// that hold vectors.
	DatabaseBytes int64 `json:"databaseBytes"`
	VectorBytes   int64 `json:"vectorBytes"`
}
