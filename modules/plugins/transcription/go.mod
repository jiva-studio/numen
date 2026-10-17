module github.com/jiva-studio/numen/modules/plugins/transcription

go 1.26.9

require (
	connectrpc.com/connect v1.21.0
	github.com/ebitengine/purego v0.11.1
	github.com/getcharzp/onnxruntime_purego v1.24.0
	github.com/hajimehoshi/go-mp3 v0.3.4
	github.com/jiva-studio/numen/modules/libs/plugin-sdk-go v0.0.0
	github.com/jiva-studio/numen/modules/libs/protocol v0.0.0
	github.com/mewkiz/flac v1.0.14
)

require (
	github.com/icza/bitio v1.1.0 // indirect
	github.com/mewkiz/pkg v0.0.0-20250417130911-3f050ff8c56d // indirect
	github.com/mewpkg/term v0.0.0-20241026122259-37a80af23985 // indirect
	github.com/up-zero/gotool v0.0.0-20260105010850-4a05e8fadd54 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/jiva-studio/numen/modules/libs/plugin-sdk-go => ../../libs/plugin-sdk-go

replace github.com/jiva-studio/numen/modules/libs/protocol => ../../libs/protocol
