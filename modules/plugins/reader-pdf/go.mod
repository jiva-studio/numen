module github.com/jiva-studio/numen/modules/plugins/reader-pdf

go 1.26.9

require (
	connectrpc.com/connect v1.21.0
	github.com/jiva-studio/numen/modules/libs/protocol v0.0.0
	github.com/klippa-app/go-pdfium v1.19.8
	golang.org/x/net v0.58.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/jolestar/go-commons-pool/v2 v2.1.2 // indirect
	github.com/tetratelabs/wazero v1.12.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/jiva-studio/numen/modules/libs/protocol => ../../libs/protocol
