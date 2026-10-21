package agentrun

import "strings"

// FailureMessage is what to tell the person when a command stopped with an error: the first line of
// its error output that starts with "error:", else the last line, or the error itself when it wrote
// none.
func FailureMessage(err error, said string) string {
	clean := strings.TrimSpace(said)
	if clean == "" {
		return err.Error()
	}
	lines := strings.Split(clean, "\n")
	for _, line := range lines {
		if text := strings.TrimSpace(line); len(text) > len("error:") && strings.EqualFold(text[:len("error:")], "error:") {
			return strings.TrimSpace(text[len("error:"):])
		}
	}
	return strings.TrimSpace(lines[len(lines)-1])
}
