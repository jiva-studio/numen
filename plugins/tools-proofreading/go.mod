module github.com/jiva-studio/numen/plugins/tools-proofreading

go 1.26.9

require (
	github.com/jiva-studio/numen/libs/plugin-sdk-go v0.0.0
	github.com/jiva-studio/numen/libs/protocol v0.0.0
	golang.org/x/text v0.43.0
)

require (
	connectrpc.com/connect v1.21.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/jiva-studio/numen/libs/notes v0.0.0
	github.com/jiva-studio/numen/libs/plugin-sdk-go/host v0.0.0
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/jiva-studio/numen/libs/plugin-sdk-go => ../../libs/plugin-sdk-go

replace github.com/jiva-studio/numen/libs/protocol => ../../libs/protocol

replace github.com/jiva-studio/numen/libs/plugin-sdk-go/host => ../../libs/plugin-sdk-go/host

replace github.com/jiva-studio/numen/libs/notes => ../../libs/notes
