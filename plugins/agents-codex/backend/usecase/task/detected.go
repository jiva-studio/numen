package task

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
)

// detectedName is the name of the connection the plugin reports while the program is found.
const detectedName = "OpenAI Codex"

// DetectConnections answers connections.detected: one connection named after the program while the program
// is found, by the lookup a run takes, and none otherwise.
func (r *Runner) DetectConnections(context.Context, connection.Call) ([]connection.DetectedConnection, error) {
	if r.FindProgram() == "" {
		return []connection.DetectedConnection{}, nil
	}
	return []connection.DetectedConnection{{Name: detectedName, Fields: map[string]string{}}}, nil
}
