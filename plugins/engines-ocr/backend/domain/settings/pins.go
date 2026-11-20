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

type recognizerPin struct {
	address string
	digest  string
}

func getRecognizers() map[string]recognizerPin {
	return map[string]recognizerPin{
		"pp-ocrv6-tiny":       {address: recogDefault, digest: "e16e242de5937ad92609223f19bc2aff3727ee40b095f996907c24749bad251b"},
		"pp-ocrv6-small":      {address: rapidOCR + "PP-OCRv6/rec/PP-OCRv6_rec_small.onnx", digest: "6f327246b50388f3c176ae304bd95767ea6dc0c9ae92153ef8cbe210b3c14884"},
		"pp-ocrv5-latin":      {address: rapidOCR + "PP-OCRv5/rec/latin_PP-OCRv5_rec_mobile.onnx", digest: "b20bd37c168a570f583afbc8cd7925603890efbcdc000a59e22c269d160b5f5a"},
		"pp-ocrv5-eslav":      {address: rapidOCR + "PP-OCRv5/rec/eslav_PP-OCRv5_rec_mobile.onnx", digest: "08705d6721849b1347d26187f15a5e362c431963a2a62bfff4feac578c489aab"},
		"pp-ocrv5-cyrillic":   {address: rapidOCR + "PP-OCRv5/rec/cyrillic_PP-OCRv5_rec_mobile.onnx", digest: "90f761b4bfcce0c8c561c0cb5c887b0971d3ec01c32164bdf7374a35b0982711"},
		"pp-ocrv5-devanagari": {address: rapidOCR + "PP-OCRv5/rec/devanagari_PP-OCRv5_rec_mobile.onnx", digest: "d6f0a906580e3fa6b324a318718f1f31f268b6ea8ef985f91c2012a37f52c91e"},
		"pp-ocrv5-greek":      {address: rapidOCR + "PP-OCRv5/rec/el_PP-OCRv5_rec_mobile.onnx", digest: "b4368bccd557123c702b7549fee6cd1e94b581337d1c9b65310f109131542b7f"},
	}
}

// FindRecognizer returns the address of the recognition model the settings name, and false for a
// name the plugin does not provision. The custom recognizer has no address.
func FindRecognizer(name string) (string, bool) {
	if name == RecognizerCustom {
		return "", true
	}
	pin, ok := getRecognizers()[name]
	return pin.address, ok
}

// FindModelDigest returns the SHA-256 pinned for a downloadable model.
func FindModelDigest(address string) (string, bool) {
	switch address {
	case layoutDefault:
		return "45bf71750b00739a41fc209f132eb104a4d6b5bb29483c9078164d8b87cf28ba", true
	case detectDefault:
		return "a431985659dc921974177a95adcfbb90fd9e51989a5e04d70d0b75f597b6e61d", true
	}
	for _, pin := range getRecognizers() {
		if pin.address == address {
			return pin.digest, true
		}
	}
	return "", false
}
