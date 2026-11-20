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

// FindModelDigest returns the SHA-256 pinned for a downloadable model file.
func FindModelDigest(address string) (string, bool) {
	switch address {
	case segmenterDefault:
		return "a4a068cd6cf1ea8355b84327595838ca748ec29a25bc91fc82e6c299ccdc5808", true
	case parakeetV3 + "encoder.int8.onnx":
		return "acfc2b4456377e15d04f0243af540b7fe7c992f8d898d751cf134c3a55fd2247", true
	case parakeetV3 + "decoder.int8.onnx":
		return "179e50c43d1a9de79c8a24149a2f9bac6eb5981823f2a2ed88d655b24248db4e", true
	case parakeetV3 + "joiner.int8.onnx":
		return "3164c13fc2821009440d20fcb5fdc78bff28b4db2f8d0f0b329101719c0948b3", true
	case parakeetV3 + "tokens.txt":
		return "d58544679ea4bc6ac563d1f545eb7d474bd6cfa467f0a6e2c1dc1c7d37e3c35d", true
	case parakeetV2 + "encoder.int8.onnx":
		return "a32b12d17bbbc309d0686fbbcc2987b5e9b8333a7da83fa6b089f0a2acd651ab", true
	case parakeetV2 + "decoder.int8.onnx":
		return "b6bb64963457237b900e496ee9994b59294526439fbcc1fecf705b31a15c6b4e", true
	case parakeetV2 + "joiner.int8.onnx":
		return "7946164367946e7f9f29a122407c3252b680dbae9a51343eb2488d057c3c43d2", true
	case parakeetV2 + "tokens.txt":
		return "ec182b70dd42113aff6c5372c75cac58c952443eb22322f57bbd7f53977d497d", true
	}
	return "", false
}
