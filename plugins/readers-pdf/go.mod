module github.com/jiva-studio/numen/plugins/readers-pdf

go 1.26.9

require (
	connectrpc.com/connect v1.21.1
	github.com/jiva-studio/numen/libs/plugin-sdk-go v0.0.0
	github.com/jiva-studio/numen/libs/protocol v0.0.0
	github.com/klippa-app/go-pdfium v1.21.4
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/jolestar/go-commons-pool/v2 v2.1.2 // indirect
	github.com/tetratelabs/wazero v1.12.0 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/jiva-studio/numen/libs/plugin-sdk-go => ../../libs/plugin-sdk-go

replace github.com/jiva-studio/numen/libs/protocol => ../../libs/protocol
