module github.com/jiva-studio/numen/modules/plugins/agent-codex

go 1.26.9

require (
	connectrpc.com/connect v1.21.1
	github.com/jiva-studio/numen/modules/libs/plugin-sdk-go v0.0.0
	github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent v0.0.0
	github.com/jiva-studio/numen/modules/libs/protocol v0.0.0
)

require google.golang.org/protobuf v1.36.12 // indirect

replace github.com/jiva-studio/numen/modules/libs/plugin-sdk-go => ../../libs/plugin-sdk-go

replace github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent => ../../libs/plugin-sdk-go/agent

replace github.com/jiva-studio/numen/modules/libs/protocol => ../../libs/protocol
