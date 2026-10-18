module github.com/jiva-studio/numen/modules/libs/plugin-sdk-go

go 1.26.9

require (
	connectrpc.com/connect v1.21.0
	github.com/jiva-studio/numen/modules/libs/protocol v0.0.0
)

require google.golang.org/protobuf v1.36.12 // indirect

replace github.com/jiva-studio/numen/modules/libs/protocol => ../protocol
