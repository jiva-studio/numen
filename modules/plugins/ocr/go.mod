module github.com/jiva-studio/numen/modules/plugins/ocr

go 1.26.9

require (
	connectrpc.com/connect v1.21.0
	github.com/jiva-studio/numen/modules/libs/protocol v0.0.0
	golang.org/x/net v0.58.0
)

require (
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/jiva-studio/numen/modules/libs/protocol => ../../libs/protocol
