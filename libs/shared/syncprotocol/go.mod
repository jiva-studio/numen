module github.com/jiva-studio/numen/libs/shared/syncprotocol

go 1.26.11

require (
	connectrpc.com/connect v1.21.1
	github.com/klauspost/compress v1.20.3
	golang.org/x/crypto v0.59.0
	golang.org/x/net v0.62.0
	google.golang.org/protobuf v1.36.12
)

require golang.org/x/sys v0.50.0 // indirect

tool (
	connectrpc.com/connect/cmd/protoc-gen-connect-go
	google.golang.org/protobuf/cmd/protoc-gen-go
)
