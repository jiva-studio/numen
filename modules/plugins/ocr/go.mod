module github.com/jiva-studio/numen/modules/plugins/ocr

go 1.26.9

require (
	connectrpc.com/connect v1.21.0
	github.com/ebitengine/purego v0.11.0
	github.com/getcharzp/go-ocr v0.0.0-20260818071741-a892e438f08d
	github.com/getcharzp/onnxruntime_purego v1.24.0
	github.com/jiva-studio/numen/modules/libs/plugin-sdk-go v0.0.0
	github.com/jiva-studio/numen/modules/libs/protocol v0.0.0
	golang.org/x/image v0.45.0
)

require (
	github.com/up-zero/gotool v0.0.0-20260120011100-d685b2532b5a // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/jiva-studio/numen/modules/libs/plugin-sdk-go => ../../libs/plugin-sdk-go

replace github.com/jiva-studio/numen/modules/libs/protocol => ../../libs/protocol
