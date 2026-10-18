module github.com/jiva-studio/numen/modules/plugins/reader-epub

go 1.26.9

require (
	connectrpc.com/connect v1.21.0
	github.com/jiva-studio/numen/modules/libs/plugin-sdk-go v0.0.0
	github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/vault v0.0.0
	github.com/jiva-studio/numen/modules/libs/protocol v0.0.0
	golang.org/x/net v0.61.0
)

require google.golang.org/protobuf v1.36.12 // indirect

replace github.com/jiva-studio/numen/modules/libs/plugin-sdk-go => ../../libs/plugin-sdk-go

replace github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/vault => ../../libs/plugin-sdk-go/vault

replace github.com/jiva-studio/numen/modules/libs/protocol => ../../libs/protocol
