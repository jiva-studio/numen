// Package api is the messages of the agent tool the plugin serves.
package api

// TranscriptCreateInput is the input of the transcript_create tool.
type TranscriptCreateInput struct {
	Path string `json:"path" jsonschema:"the audio or video recording, relative to the vault root"`
}

// TranscriptCreateOutput is the output of the transcript_create tool.
type TranscriptCreateOutput struct {
	TaskID  string `json:"taskId" jsonschema:"the id of the transcription in the list of tasks"`
	Path    string `json:"path" jsonschema:"the recording; its transcript can be read and searched"`
	State   string `json:"state" jsonschema:"started; running when a transcription of this recording is already in progress and no second one was started; or done when the recording is already transcribed and nothing was started"`
	Version string `json:"version,omitempty" jsonschema:"with the state done, the id of the live text version, the version that is read: the transcript, or a version made from it such as a proofread"`
}
