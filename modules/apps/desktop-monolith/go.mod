module github.com/jiva-studio/numen/modules/apps/desktop-monolith

go 1.26.9

require (
	github.com/jiva-studio/numen/modules/apps/desktop v0.0.0
	github.com/jiva-studio/numen/modules/libs/protocol v0.0.0
	github.com/jiva-studio/numen/modules/plugins/reader-epub v0.0.0
	github.com/jiva-studio/numen/modules/plugins/reader-pdf v0.0.0
	github.com/quasilyte/go-ruleguard/dsl v0.3.23
)

require (
	connectrpc.com/connect v1.21.0 // indirect
	github.com/adrg/xdg v0.5.3 // indirect
	github.com/coder/websocket v1.8.14 // indirect
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.11.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jiva-studio/numen/modules/libs/core v0.0.0 // indirect
	github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/vault v0.0.0 // indirect
	github.com/jolestar/go-commons-pool/v2 v2.1.2 // indirect
	github.com/klippa-app/go-pdfium v1.19.8 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/modelcontextprotocol/go-sdk v1.8.0 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/open-spaced-repetition/go-fsrs/v3 v3.3.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rjeczalik/notify v0.9.3 // indirect
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/tetratelabs/wazero v1.12.0 // indirect
	github.com/wailsapp/wails/v3 v3.0.0-beta.29 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	github.com/zalando/go-keyring v0.2.8 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.75.7 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
	modernc.org/sqlite v1.59.0 // indirect
)

replace github.com/jiva-studio/numen/modules/apps/desktop => ../desktop

replace github.com/jiva-studio/numen/modules/libs/core => ../../libs/core

replace github.com/jiva-studio/numen/modules/libs/protocol => ../../libs/protocol

replace github.com/jiva-studio/numen/modules/plugins/reader-epub => ../../plugins/reader-epub

replace github.com/jiva-studio/numen/modules/plugins/reader-pdf => ../../plugins/reader-pdf

replace github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/vault => ../../libs/plugin-sdk-go/vault
