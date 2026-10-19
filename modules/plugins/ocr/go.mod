module github.com/jiva-studio/numen/modules/plugins/ocr

go 1.26.9

require (
	connectrpc.com/connect v1.21.1
	github.com/getcharzp/go-ocr v0.0.0-20260818071741-a892e438f08d
	github.com/getcharzp/onnxruntime_purego v1.25.0
	github.com/jiva-studio/numen/modules/libs/plugin-sdk-go v0.0.0
	github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models v0.0.0
	github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx v0.0.0
	github.com/jiva-studio/numen/modules/libs/protocol v0.0.0
	golang.org/x/image v0.47.0
)

require (
	github.com/ebitengine/purego v0.11.2 // indirect
	github.com/up-zero/gotool v0.0.0-20260120011100-d685b2532b5a // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/jiva-studio/numen/modules/libs/plugin-sdk-go => ../../libs/plugin-sdk-go

replace github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models => ../../libs/plugin-sdk-go/models

replace github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx => ../../libs/plugin-sdk-go/onnx

replace github.com/jiva-studio/numen/modules/libs/protocol => ../../libs/protocol
