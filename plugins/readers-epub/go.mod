module github.com/jiva-studio/numen/plugins/readers-epub

go 1.26.9

require (
	connectrpc.com/connect v1.21.1
	github.com/jiva-studio/numen/libs/notes v0.0.0
	github.com/jiva-studio/numen/libs/plugin-sdk-go v0.0.0
	github.com/jiva-studio/numen/libs/plugin-sdk-go/host v0.0.0
	github.com/jiva-studio/numen/libs/protocol v0.0.0
	golang.org/x/net v0.61.0
)

require (
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/jiva-studio/numen/libs/notes => ../../libs/notes
	github.com/jiva-studio/numen/libs/plugin-sdk-go => ../../libs/plugin-sdk-go
	github.com/jiva-studio/numen/libs/plugin-sdk-go/host => ../../libs/plugin-sdk-go/host
	github.com/jiva-studio/numen/libs/protocol => ../../libs/protocol
)
