module github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx

go 1.26.9

require (
	github.com/ebitengine/purego v0.11.2
	github.com/getcharzp/onnxruntime_purego v1.25.0
	github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models v0.0.0
)

require github.com/up-zero/gotool v0.0.0-20260105010850-4a05e8fadd54 // indirect

replace github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models => ../models

