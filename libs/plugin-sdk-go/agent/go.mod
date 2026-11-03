module github.com/jiva-studio/numen/libs/plugin-sdk-go/agent

go 1.26.10

require (
	connectrpc.com/connect v1.21.1
	github.com/jiva-studio/numen/libs/plugin-sdk-go v0.0.0
	github.com/jiva-studio/numen/libs/protocol v0.0.0
	google.golang.org/protobuf v1.36.12
)

replace github.com/jiva-studio/numen/libs/protocol => ../../protocol

replace github.com/jiva-studio/numen/libs/plugin-sdk-go => ../
