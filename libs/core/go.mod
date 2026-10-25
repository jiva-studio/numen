module github.com/jiva-studio/numen/libs/core

go 1.26.9

require (
	connectrpc.com/connect v1.21.1
	github.com/ebitengine/purego v0.11.2
	github.com/rjeczalik/notify v0.9.3
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06
	github.com/zalando/go-keyring v0.2.8
	golang.org/x/sync v0.24.0
	golang.org/x/text v0.43.0
	google.golang.org/protobuf v1.36.12
	gopkg.in/yaml.v3 v3.0.1
	modernc.org/sqlite v1.60.1
	pgregory.net/rapid v1.3.0
)

require (
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jiva-studio/numen/libs/notes v0.0.0
	github.com/jiva-studio/numen/libs/protocol v0.0.0
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/sys v0.49.0
	modernc.org/libc v1.77.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)

replace github.com/jiva-studio/numen/libs/protocol => ../protocol

replace github.com/jiva-studio/numen/libs/notes => ../notes
