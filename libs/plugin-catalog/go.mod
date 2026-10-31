module github.com/jiva-studio/numen/libs/plugin-catalog

go 1.26.9

require (
	github.com/jiva-studio/numen/libs/notes v0.0.0
	github.com/jiva-studio/numen/libs/protocol v0.0.0
	google.golang.org/protobuf v1.36.12
)

require golang.org/x/text v0.43.0 // indirect

replace (
	github.com/jiva-studio/numen/libs/notes => ../notes
	github.com/jiva-studio/numen/libs/protocol => ../protocol
)
