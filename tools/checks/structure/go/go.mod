module github.com/jiva-studio/numen/tools/checks/structure/go

go 1.26.11

require (
	github.com/jiva-studio/numen/libs/plugin/catalog v0.0.0
	golang.org/x/mod v0.42.0
	golang.org/x/tools v0.52.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cel.dev/cel-go v0.32.0 // indirect
	cel.dev/expr v0.25.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/jiva-studio/numen/libs/atomicfile v0.0.0 // indirect
	github.com/jiva-studio/numen/libs/core/protocol v0.0.0 // indirect
	github.com/jiva-studio/numen/libs/note v0.0.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20240823005443-9b4947da3948 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240826202546-f6391c0de4c7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240826202546-f6391c0de4c7 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace (
	github.com/jiva-studio/numen/libs/core/protocol => ../../../../libs/core/protocol
	github.com/jiva-studio/numen/libs/note => ../../../../libs/note
	github.com/jiva-studio/numen/libs/plugin/catalog => ../../../../libs/plugin/catalog
)

replace github.com/jiva-studio/numen/libs/atomicfile => ../../../../libs/atomicfile
