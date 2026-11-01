module github.com/jiva-studio/numen/libs/plugin-sdk-go

go 1.26.9

require (
	connectrpc.com/connect v1.21.1
	github.com/google/jsonschema-go v0.4.3
	github.com/jiva-studio/numen/libs/plugin-catalog v0.0.0
	github.com/jiva-studio/numen/libs/protocol v0.0.0
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/jiva-studio/numen/libs/notes v0.0.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)

replace (
	github.com/jiva-studio/numen/libs/notes => ../notes
	github.com/jiva-studio/numen/libs/plugin-catalog => ../plugin-catalog
	github.com/jiva-studio/numen/libs/protocol => ../protocol
)
