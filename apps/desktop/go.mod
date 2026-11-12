module github.com/jiva-studio/numen/apps/desktop

go 1.26.11

require (
	connectrpc.com/connect v1.21.1 // indirect
	github.com/ebitengine/purego v0.11.3 // indirect
	github.com/rjeczalik/notify v0.9.3 // indirect
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06 // indirect
	github.com/wailsapp/wails/v3 v3.2.0-nightly.20261101
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/sqlite v1.61.1 // indirect
)

require (
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

require (
	cel.dev/cel-go v0.32.0 // indirect
	cel.dev/expr v0.25.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/coder/websocket v1.8.14 // indirect
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/godbus/dbus/v5 v5.2.4
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jiva-studio/numen/libs/core/protocol v0.0.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/modelcontextprotocol/go-sdk v1.8.0 // indirect
	github.com/ncruces/go-strftime v1.1.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20260410095643-746e56fc9e2f // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sys v0.49.0
	golang.org/x/term v0.46.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240826202546-f6391c0de4c7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240826202546-f6391c0de4c7 // indirect
	modernc.org/libc v1.79.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)

replace github.com/jiva-studio/numen/libs/core/protocol => ../../libs/core/protocol

require (
	github.com/jiva-studio/numen/libs/core/kernel v0.0.0
	github.com/jiva-studio/numen/libs/atomicfile v0.0.0
	github.com/jiva-studio/numen/libs/note v0.0.0
	github.com/jiva-studio/numen/libs/plugin/catalog v0.0.0
	github.com/zalando/go-keyring v0.2.8
)

replace github.com/jiva-studio/numen/libs/plugin/catalog => ../../libs/plugin/catalog

replace github.com/jiva-studio/numen/libs/core/kernel => ../../libs/core/kernel

replace github.com/jiva-studio/numen/libs/note => ../../libs/note

replace github.com/jiva-studio/numen/libs/atomicfile => ../../libs/atomicfile
