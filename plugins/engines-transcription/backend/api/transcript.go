// Package api is the messages of the agent tool the plugin serves.
package api

type TranscriptCreateInput struct {
	Path string `json:"path" jsonschema:"the audio or video recording, relative to the vault root"`
}

type TranscriptCreateOutput struct {
	TaskID  string `json:"taskId"            jsonschema:"the id of the transcription in the list of tasks"`
	Path    string `json:"path"              jsonschema:"the recording; its transcript can be read and searched"`
	State   string `json:"state"             jsonschema:"started; running when the recording is already being transcribed (no second run starts); or done when it is already transcribed"`
	Version string `json:"version,omitempty" jsonschema:"with the state done, the id of the live text version, the version that is read: the transcript, or a version made from it such as a proofread"`
}
