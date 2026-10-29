package engine

// Default addresses for the transcription models.
const (
	parakeetV3       = "https://huggingface.co/csukuangfj/sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8/resolve/2bda32ec70b097a55adaa07d9a7173915b43cc78/"
	parakeetV2       = "https://huggingface.co/csukuangfj/sherpa-onnx-nemo-parakeet-tdt-0.6b-v2-int8/resolve/1ab9323565ddb038682214b292f588070a538ce2/"
	segmenterDefault = "https://huggingface.co/onnx-community/silero-vad/resolve/e71cae966052b992a7eca6b17738916ce0eca4ec/onnx/model.onnx"
)

// LanguageDetected is the language of a model that detects the language of its input and
// cannot be given one.
const LanguageDetected = "auto"

// models are the Parakeet exports the plugin provisions, by the name the settings give each. Both
// are FastConformer-TDT transducers over 128 mel features and differ in their vocabulary: v3 recognises
// 25 European languages and detects which, v2 recognises English.
var models = []ParakeetModel{
	{Name: "parakeet-tdt-0.6b-v3-int8", Repo: parakeetV3, Languages: []string{LanguageDetected}},
	{Name: "parakeet-tdt-0.6b-v2-int8", Repo: parakeetV2, Languages: []string{"en"}},
}

// FindModel returns the model the settings name.
func FindModel(name string) (ParakeetModel, bool) {
	for _, one := range models {
		if one.Name == name {
			return one, true
		}
	}
	return ParakeetModel{}, false
}
