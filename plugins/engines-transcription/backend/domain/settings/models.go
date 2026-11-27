package settings

// GetModels returns the speech models the plugin offers, each with the ids of the files its manifest
// declares.
func GetModels() []ParakeetModel {
	return []ParakeetModel{
		{
			Name: "parakeet-tdt-0.6b-v3-int8", Repo: "",
			Encoder: "", Decoder: "", Joiner: "", Tokens: "",
			Dependencies: ParakeetFiles{
				Encoder: "parakeet-v3-encoder", Decoder: "parakeet-v3-decoder", Joiner: "parakeet-v3-joiner", Tokens: "parakeet-v3-tokens",
			},
		},
		{
			Name: "parakeet-tdt-0.6b-v2-int8", Repo: "",
			Encoder: "", Decoder: "", Joiner: "", Tokens: "",
			Dependencies: ParakeetFiles{
				Encoder: "parakeet-v2-encoder", Decoder: "parakeet-v2-decoder", Joiner: "parakeet-v2-joiner", Tokens: "parakeet-v2-tokens",
			},
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
		Name: "", Repo: "", Encoder: "", Decoder: "", Joiner: "", Tokens: "", Dependencies: ParakeetFiles{Encoder: "", Decoder: "", Joiner: "", Tokens: ""},
	}, false
}
