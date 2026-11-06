package engine

import "strings"

// The models the defaults fetch, each at a fixed revision.
const (
	layoutDefault = "https://huggingface.co/PaddlePaddle/PP-DocLayoutV3_onnx/resolve/46bbdf188bb0a772c08aed74882ce7e51a8f1ea6/inference.onnx"
	detectDefault = "https://huggingface.co/PaddlePaddle/PP-OCRv5_mobile_det_onnx/resolve/e6f4fa85f00e168c862bc462aebca69eef9b3d3d/inference.onnx"
	rapidOCR      = "https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/4e9666f0fbf5fac850d30e530bf0a2e0d886aa78/onnx/"
)

// Recogniser names the settings give.
const (
	// RecogniserCustom is the recogniser the user selects by its file.
	RecogniserCustom = "custom"
	// RecogniserMultilingual reads every line with the Latin, Cyrillic, Greek and PP-OCRv6 tiny models.
	RecogniserMultilingual = "multilingual"
	// RecogniserCustomFallback reads while the custom model has no file: the manifest provisions it
	// for the custom option.
	RecogniserCustomFallback = "pp-ocrv6-tiny"
)

// RecogniserFile is one recognition model the plugin provisions: the dependency id, the address it
// is fetched from, and the variable and file name the host gives it in.
type RecogniserFile struct {
	ID       string `json:"id"`
	Address  string `json:"address"`
	EnvVar   string `json:"env_var"`
	FileName string `json:"file_name"`
}

func newRecogniserFile(id, path string) RecogniserFile {
	return RecogniserFile{
		ID:       id,
		Address:  rapidOCR + path,
		EnvVar:   "NUMEN_DEPENDENCY_" + strings.ToUpper(strings.ReplaceAll(id, "-", "_")),
		FileName: id + ".onnx",
	}
}

// getRecognisers returns the recognition models of each name the settings give, in the order a tie
// between their readings of one line is broken in. Each model carries its alphabet in its metadata.
func getRecognisers() map[string][]RecogniserFile {
	tiny := newRecogniserFile("rec-pp-ocrv6-tiny", "PP-OCRv6/rec/PP-OCRv6_rec_tiny.onnx")
	latin := newRecogniserFile("rec-pp-ocrv5-latin", "PP-OCRv5/rec/latin_PP-OCRv5_rec_mobile.onnx")
	cyrillic := newRecogniserFile("rec-pp-ocrv5-cyrillic", "PP-OCRv5/rec/cyrillic_PP-OCRv5_rec_mobile.onnx")
	greek := newRecogniserFile("rec-pp-ocrv5-greek", "PP-OCRv5/rec/el_PP-OCRv5_rec_mobile.onnx")
	return map[string][]RecogniserFile{
		RecogniserMultilingual: {latin, cyrillic, greek, tiny},
		"pp-ocrv6-tiny":        {tiny},
		"pp-ocrv6-small":       {newRecogniserFile("rec-pp-ocrv6-small", "PP-OCRv6/rec/PP-OCRv6_rec_small.onnx")},
		"pp-ocrv5-latin":       {latin},
		"pp-ocrv5-eslav":       {newRecogniserFile("rec-pp-ocrv5-eslav", "PP-OCRv5/rec/eslav_PP-OCRv5_rec_mobile.onnx")},
		"pp-ocrv5-cyrillic":    {cyrillic},
		"pp-ocrv5-devanagari":  {newRecogniserFile("rec-pp-ocrv5-devanagari", "PP-OCRv5/rec/devanagari_PP-OCRv5_rec_mobile.onnx")},
		"pp-ocrv5-greek":       {greek},
	}
}

// FindRecognisers returns the recognition models the setting value names, in the order a tie is
// broken in, and false for a value the plugin does not provision. The custom model names none.
func FindRecognisers(name string) ([]RecogniserFile, bool) {
	if name == RecogniserCustom {
		return nil, true
	}
	files, ok := getRecognisers()[name]
	return files, ok
}
