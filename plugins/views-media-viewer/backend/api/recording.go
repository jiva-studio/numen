// Package api holds the messages of the audio recorder.
package api

const Service = "md.numen.views.media.viewer"
const SaveRecording = "SaveRecording"

// SaveRequest carries the captured audio and its destination.
type SaveRequest struct {
	StartedAt string `json:"startedAt"`
	VaultID   string `json:"vaultId"`
	Folder    string `json:"folder"`
	MediaType string `json:"mediaType"`
	Content   []byte `json:"content"`
}

// SaveResponse names the vault file containing the recording.
type SaveResponse struct {
	Path string `json:"path"`
}
