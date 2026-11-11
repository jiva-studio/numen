module github.com/jiva-studio/numen/tools/checks/structure/go

go 1.26.11

require (
	github.com/jiva-studio/numen/libs/plugin/catalog v0.0.0
	golang.org/x/tools v0.52.0
	gopkg.in/yaml.v3 v3.0.1
)

replace (
	github.com/jiva-studio/numen/libs/core/protocol => ../../../../libs/core/protocol
	github.com/jiva-studio/numen/libs/note => ../../../../libs/note
	github.com/jiva-studio/numen/libs/plugin/catalog => ../../../../libs/plugin/catalog
)
