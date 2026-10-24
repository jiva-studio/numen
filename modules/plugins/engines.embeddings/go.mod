module github.com/jiva-studio/numen/modules/plugins/engines.embeddings

go 1.26.9

require (
	connectrpc.com/connect v1.21.1
	github.com/getcharzp/onnxruntime_purego v1.25.0
	github.com/gomlx/go-huggingface v0.4.3
	github.com/jiva-studio/numen/modules/libs/plugin-sdk-go v0.0.0
	github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx v0.0.0
	github.com/jiva-studio/numen/modules/libs/protocol v0.0.0
)

require (
	github.com/ebitengine/purego v0.11.2 // indirect
	github.com/gofrs/flock v0.13.0 // indirect
	github.com/gomlx/compute v0.1.6 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/up-zero/gotool v0.0.0-20260105010850-4a05e8fadd54 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/jiva-studio/numen/modules/libs/plugin-sdk-go => ../../libs/plugin-sdk-go

replace github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx => ../../libs/plugin-sdk-go/onnx

replace github.com/jiva-studio/numen/modules/libs/protocol => ../../libs/protocol
