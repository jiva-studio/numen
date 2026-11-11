module github.com/jiva-studio/numen/libs/plugin/sdk-go/agent

go 1.26.11

require (
	connectrpc.com/connect v1.21.1
	github.com/jiva-studio/numen/libs/plugin/sdk-go v0.0.0
	github.com/jiva-studio/numen/libs/core/protocol v0.0.0
	google.golang.org/protobuf v1.36.12
)

replace github.com/jiva-studio/numen/libs/core/protocol => ../../../core/protocol

replace github.com/jiva-studio/numen/libs/plugin/sdk-go => ../
