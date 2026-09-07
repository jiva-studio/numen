module github.com/jiva-studio/numen/modules/apps/desktop

go 1.27

require (
	connectrpc.com/connect v1.20.0 // indirect
	github.com/adrg/xdg v0.5.3 // indirect
	github.com/coder/websocket v1.8.14 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.11.0 // indirect
	github.com/getcharzp/go-ocr v0.0.0-20260818071741-a892e438f08d // indirect
	github.com/getcharzp/onnxruntime_purego v1.24.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/gofrs/flock v0.13.0 // indirect
	github.com/gomlx/compute v0.1.7 // indirect
	github.com/gomlx/go-huggingface v0.4.4 // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hajimehoshi/go-mp3 v0.3.4 // indirect
	github.com/icza/bitio v1.1.0 // indirect
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
	github.com/jiva-studio/numen/modules/libs/protocol v0.0.0 // indirect
	github.com/jolestar/go-commons-pool/v2 v2.1.2 // indirect
	github.com/klippa-app/go-pdfium v1.19.8 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/mewkiz/flac v1.0.14 // indirect
	github.com/mewkiz/pkg v0.0.0-20250417130911-3f050ff8c56d // indirect
	github.com/mewpkg/term v0.0.0-20241026122259-37a80af23985 // indirect
	github.com/modelcontextprotocol/go-sdk v1.7.0 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/open-spaced-repetition/go-fsrs/v3 v3.3.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rjeczalik/notify v0.9.3 // indirect
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/tetratelabs/wazero v1.12.0 // indirect
	github.com/up-zero/gotool v0.0.0-20260120011100-d685b2532b5a // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/image v0.45.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.49.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.75.6 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
	modernc.org/sqlite v1.58.0 // indirect
)

replace github.com/jiva-studio/numen/modules/libs/protocol => ../../libs/protocol

require (
	github.com/jiva-studio/numen/modules/libs/core v0.0.0
	github.com/quasilyte/go-ruleguard/dsl v0.3.23
	github.com/wailsapp/wails/v3 v3.0.0-beta.16
)

replace github.com/jiva-studio/numen/modules/libs/core => ../../libs/core
