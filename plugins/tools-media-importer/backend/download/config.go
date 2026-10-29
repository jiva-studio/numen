package download

import (
	"maps"
	"os"
	"slices"
)

// Config lists the programs that this machine uses to access a video site.
type Config struct {
	Video Tool
	Sound Tool
}

// Tool is a program this machine holds.
type Tool struct {
	Command     []string
	Arguments   []string
	Environment map[string]string
}

func (t Tool) getEnvironment() []string {
	if len(t.Environment) == 0 {
		return nil
	}
	out := os.Environ()
	for _, name := range slices.Sorted(maps.Keys(t.Environment)) {
		out = append(out, name+"="+t.Environment[name])
	}
	return out
}

// Defaults is the programs PATH and the host name.
func Defaults() Config { return Config{} }
