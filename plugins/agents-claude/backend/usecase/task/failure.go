package task

import "strings"

func getFailureMessage(err error, output string) string {
	clean := strings.TrimSpace(output)
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
