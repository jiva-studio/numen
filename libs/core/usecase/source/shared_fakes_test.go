package source

import "strings"

const recordingPath = "library/talk.mp3"

// here reads them: what the recording says comes from the model, and the model
// is a fake.
func newRecording(words []string) []byte {
	return []byte("a recording of " + strings.Join(words, "|"))
}
