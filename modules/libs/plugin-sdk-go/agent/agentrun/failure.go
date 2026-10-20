package agentrun

import "strings"

// FailureMessage is what to tell the person when a command stopped with an error: the last line of
// its error output, or the error itself when it wrote none.
func FailureMessage(err error, said string) string {
	if trimmed := strings.TrimSpace(said); trimmed != "" {
		lines := strings.Split(trimmed, "\n")
		return strings.TrimSpace(lines[len(lines)-1])
	}
	return err.Error()
}
