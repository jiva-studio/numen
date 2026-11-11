package settings

import "strconv"

// ParseRecognizerSettings returns current changed by the plugin settings: a positive resolution
// replaces the DPI, a known model replaces the model, and the custom model files are read in any case.
func ParseRecognizerSettings(current RecognizerModel, values map[string]string) RecognizerModel {
	next := current
	if dpi, err := strconv.Atoi(values["dpi"]); err == nil && dpi > 0 {
		next.DPI = dpi
	}
	if address, ok := FindRecognizer(values["model"]); ok {
		next.Name = address
		if address == "" {
			next.Name = RecognizerCustom
		}
	}
	next.Path, next.Dict = values["customModelPath"], values["customDictPath"]
	return next
}
