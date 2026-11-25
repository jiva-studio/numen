module github.com/jiva-studio/numen/libs/plugin/sdk-go

go 1.26.11

require (
	connectrpc.com/connect v1.21.1
	github.com/google/jsonschema-go v0.4.3
	github.com/jiva-studio/numen/libs/shared/atomicfile v0.0.0
	github.com/jiva-studio/numen/libs/shared/identity v0.0.0
	github.com/jiva-studio/numen/libs/shared/note v0.0.0
	github.com/jiva-studio/numen/libs/shared/protocol v0.0.0
	github.com/jiva-studio/numen/libs/shared/rerank v0.0.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240826202546-f6391c0de4c7
	google.golang.org/protobuf v1.36.12
)

require (
	golang.org/x/text v0.43.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/jiva-studio/numen/libs/shared/protocol => ../../shared/protocol

replace github.com/jiva-studio/numen/libs/shared/note => ../../shared/note

replace github.com/jiva-studio/numen/libs/shared/rerank => ../../shared/rerank

replace github.com/jiva-studio/numen/libs/shared/atomicfile => ../../../libs/shared/atomicfile

replace github.com/jiva-studio/numen/libs/shared/identity => ../../shared/identity
