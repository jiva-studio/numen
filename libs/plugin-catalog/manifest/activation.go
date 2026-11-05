package manifest

import (
	"strings"
	"unicode"
)

const (
	// ActivationEventStartup is the activation event the host emits once, when the editor window opens.
	ActivationEventStartup = "onStartup"

	// ActivationEventVectorsReady is the activation event of a plugin started by `onVectorsReady:<vault id>`,
	// which the host emits after an indexing pass leaves no vector of the vault pending.
	ActivationEventVectorsReady = "onVectorsReady"

	// ActivationEventFilePrefix followed by a file extension is the activation event the host emits
	// when a window lists a file with that extension.
	ActivationEventFilePrefix = "onFile:"
)

// isActivationEvent reports whether the host emits the event a manifest declares.
func isActivationEvent(event string) bool {
	if event == ActivationEventStartup || event == ActivationEventVectorsReady {
		return true
	}
	extension, ok := strings.CutPrefix(event, ActivationEventFilePrefix)
	return ok && isFileExtension(extension)
}

// isFileExtension reports whether the text is a file extension without its dot: letters and digits.
func isFileExtension(extension string) bool {
	if extension == "" {
		return false
	}
	return !strings.ContainsFunc(extension, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// IsActivatedBy reports whether the plugin declares the event the host emits. ActivationEventVectorsReady
// matches the vectors-ready event of every vault; any other declared event matches itself.
func (m Manifest) IsActivatedBy(event string) bool {
	for _, declared := range m.ActivationEvents {
		if declared == ActivationEventVectorsReady {
			if vaultID, ok := strings.CutPrefix(event, ActivationEventVectorsReady+":"); ok && vaultID != "" {
				return true
			}
			continue
		}
		if declared == event {
			return true
		}
	}
	return false
}
