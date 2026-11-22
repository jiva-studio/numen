module github.com/jiva-studio/numen/libs/shared/syncprotocol

go 1.26.11

require (
	connectrpc.com/connect v1.21.1
	github.com/jiva-studio/numen/libs/shared/identity v0.0.0
	google.golang.org/protobuf v1.36.12
)

tool (
	connectrpc.com/connect/cmd/protoc-gen-connect-go
	google.golang.org/protobuf/cmd/protoc-gen-go
)
