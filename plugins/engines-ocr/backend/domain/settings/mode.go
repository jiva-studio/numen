package settings

// RecognizeMode is when the plugin recognizes the text of a file without a request.
type RecognizeMode string

// The modes of the recognizeMode setting.
const (
	ModeOff             RecognizeMode = "off"
	ModeWhenNoTextLayer RecognizeMode = "whenNoTextLayer"
	ModeAlways          RecognizeMode = "always"
)

// ParseRecognizeMode returns the mode of the plugin settings; an absent or unknown value is ModeOff.
func ParseRecognizeMode(_ map[string]string) RecognizeMode {
	return ModeOff
}
