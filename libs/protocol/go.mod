module github.com/jiva-studio/numen/libs/protocol

go 1.26.9

require (
	connectrpc.com/connect v1.21.1
	google.golang.org/protobuf v1.36.12
)

tool (
	connectrpc.com/connect/cmd/protoc-gen-connect-go
	google.golang.org/protobuf/cmd/protoc-gen-go
)
