package settings

// The ids under which the manifest declares the models the defaults use.
const (
	layoutDefault = "pp-doclayout"
	detectDefault = "pp-ocr-det"
	recogDefault  = "rec-pp-ocrv6-tiny"
)

// RecognizerCustom is the recognizer the user selects by its file.
const RecognizerCustom = "custom"

func getRecognizers() map[string]string {
	return map[string]string{
		"pp-ocrv6-tiny":       recogDefault,
		"pp-ocrv6-small":      "rec-pp-ocrv6-small",
		"pp-ocrv5-latin":      "rec-pp-ocrv5-latin",
		"pp-ocrv5-eslav":      "rec-pp-ocrv5-eslav",
		"pp-ocrv5-cyrillic":   "rec-pp-ocrv5-cyrillic",
		"pp-ocrv5-devanagari": "rec-pp-ocrv5-devanagari",
		"pp-ocrv5-greek":      "rec-pp-ocrv5-greek",
	}
}

// FindRecognizer returns the id under which the manifest declares the recognition model the settings
// name, and false for a name the plugin does not provision. The custom recognizer has no id.
func FindRecognizer(name string) (string, bool) {
	if name == RecognizerCustom {
		return "", true
	}
	id, ok := getRecognizers()[name]
	return id, ok
}
