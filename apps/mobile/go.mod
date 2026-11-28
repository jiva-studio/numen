module github.com/jiva-studio/numen/apps/mobile

go 1.26.11

require (
	github.com/jiva-studio/numen/libs/core/kernel v0.0.0
	github.com/jiva-studio/numen/libs/shared/atomicfile v0.0.0 // indirect
	github.com/jiva-studio/numen/libs/shared/identity v0.0.0 // indirect
	github.com/jiva-studio/numen/libs/shared/protocol v0.0.0
	github.com/jiva-studio/numen/libs/shared/rerank v0.0.0 // indirect
	golang.org/x/mobile v0.0.0-20260821190718-4776eadac327
)

require (
	cel.dev/cel-go v0.33.2 // indirect
	cel.dev/expr v0.25.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/ebitengine/purego v0.11.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jiva-studio/numen/libs/plugin/catalog v0.0.0 // indirect
	github.com/jiva-studio/numen/libs/shared/note v0.0.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.1.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rjeczalik/notify v0.9.3 // indirect
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20240823005443-9b4947da3948 // indirect
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/tools v0.52.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240826202546-f6391c0de4c7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240826202546-f6391c0de4c7 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.80.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
	modernc.org/sqlite v1.62.0 // indirect
)

replace github.com/jiva-studio/numen/libs/core/kernel => ../../libs/core/kernel

replace github.com/jiva-studio/numen/libs/shared/protocol => ../../libs/shared/protocol

replace github.com/jiva-studio/numen/libs/shared/note => ../../libs/shared/note

replace github.com/jiva-studio/numen/libs/plugin/catalog => ../../libs/plugin/catalog

replace github.com/jiva-studio/numen/libs/shared/atomicfile => ../../libs/shared/atomicfile

require (
	connectrpc.com/connect v1.21.1
	github.com/jiva-studio/numen/libs/plugin/sdk-go v0.0.0
	github.com/jiva-studio/numen/plugins/tools-updater v0.0.0
)

replace github.com/jiva-studio/numen/libs/plugin/sdk-go => ../../libs/plugin/sdk-go

replace github.com/jiva-studio/numen/plugins/tools-updater => ../../plugins/tools-updater

replace github.com/jiva-studio/numen/libs/shared/identity => ../../libs/shared/identity
