package settings

// Default addresses for the transcription models.
const (
	parakeetV3       = "https://huggingface.co/csukuangfj/sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8/resolve/2bda32ec70b097a55adaa07d9a7173915b43cc78/"
	parakeetV2       = "https://huggingface.co/csukuangfj/sherpa-onnx-nemo-parakeet-tdt-0.6b-v2-int8/resolve/1ab9323565ddb038682214b292f588070a538ce2/"
	segmenterDefault = "https://huggingface.co/onnx-community/silero-vad/resolve/e71cae966052b992a7eca6b17738916ce0eca4ec/onnx/model.onnx"
)

func GetModels() []ParakeetModel {
	return []ParakeetModel{
		{
			Name: "parakeet-tdt-0.6b-v3-int8", Repo: parakeetV3,
			Encoder: "", Decoder: "", Joiner: "", Tokens: "",
		},
		{
			Name: "parakeet-tdt-0.6b-v2-int8", Repo: parakeetV2,
			Encoder: "", Decoder: "", Joiner: "", Tokens: "",
		},
	}
}

// FindModel returns the model the settings name.
func FindModel(name string) (ParakeetModel, bool) {
	for _, one := range GetModels() {
		if one.Name == name {
			return one, true
		}
	}
	return ParakeetModel{
		Name: "", Repo: "", Encoder: "", Decoder: "", Joiner: "", Tokens: "",
	}, false
}
