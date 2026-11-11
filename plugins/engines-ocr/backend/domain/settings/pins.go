package settings

// The models the defaults fetch, each at a fixed revision.
const (
	layoutDefault = "https://huggingface.co/PaddlePaddle/PP-DocLayoutV3_onnx/resolve/46bbdf188bb0a772c08aed74882ce7e51a8f1ea6/inference.onnx"
	detectDefault = "https://huggingface.co/PaddlePaddle/PP-OCRv5_mobile_det_onnx/resolve/e6f4fa85f00e168c862bc462aebca69eef9b3d3d/inference.onnx"
	rapidOCR      = "https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/4e9666f0fbf5fac850d30e530bf0a2e0d886aa78/onnx/"
	recogDefault  = rapidOCR + "PP-OCRv6/rec/PP-OCRv6_rec_tiny.onnx"
)

// RecognizerCustom is the recognizer the user selects by its file.
const RecognizerCustom = "custom"

func getRecognizers() map[string]string {
	return map[string]string{
		"pp-ocrv6-tiny":       recogDefault,
		"pp-ocrv6-small":      rapidOCR + "PP-OCRv6/rec/PP-OCRv6_rec_small.onnx",
		"pp-ocrv5-latin":      rapidOCR + "PP-OCRv5/rec/latin_PP-OCRv5_rec_mobile.onnx",
		"pp-ocrv5-eslav":      rapidOCR + "PP-OCRv5/rec/eslav_PP-OCRv5_rec_mobile.onnx",
		"pp-ocrv5-cyrillic":   rapidOCR + "PP-OCRv5/rec/cyrillic_PP-OCRv5_rec_mobile.onnx",
		"pp-ocrv5-devanagari": rapidOCR + "PP-OCRv5/rec/devanagari_PP-OCRv5_rec_mobile.onnx",
		"pp-ocrv5-greek":      rapidOCR + "PP-OCRv5/rec/el_PP-OCRv5_rec_mobile.onnx",
	}
}

// FindRecognizer returns the address of the recognition model the settings name, and false for a
// name the plugin does not provision. The custom recognizer has no address.
func FindRecognizer(name string) (string, bool) {
	if name == RecognizerCustom {
		return "", true
	}
	address, ok := getRecognizers()[name]
	return address, ok
}
