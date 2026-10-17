package engine

import "github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"

// The models the defaults fetch, each at a fixed revision and with the sha256
// its bytes must hold.
const (
	layoutDefault = "https://huggingface.co/PaddlePaddle/PP-DocLayoutV3_onnx/resolve/46bbdf188bb0a772c08aed74882ce7e51a8f1ea6/inference.onnx"
	detectDefault = "https://huggingface.co/PaddlePaddle/PP-OCRv5_mobile_det_onnx/resolve/e6f4fa85f00e168c862bc462aebca69eef9b3d3d/inference.onnx"
	recogDefault  = "https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/4e9666f0fbf5fac850d30e530bf0a2e0d886aa78/onnx/PP-OCRv6/rec/PP-OCRv6_rec_tiny.onnx"
)

var pinnedSums = models.Pins{
	layoutDefault: "45bf71750b00739a41fc209f132eb104a4d6b5bb29483c9078164d8b87cf28ba",
	detectDefault: "a431985659dc921974177a95adcfbb90fd9e51989a5e04d70d0b75f597b6e61d",
	recogDefault:  "e16e242de5937ad92609223f19bc2aff3727ee40b095f996907c24749bad251b",
}
