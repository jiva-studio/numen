package load

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Wire is the module the schema is generated into.
const Wire = "github.com/jiva-studio/numen/libs/core/protocol"

func ListShippedPlatforms() []struct{ goos, goarch string } {
	return []struct{ goos, goarch string }{
		{"linux", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
		{"android", "arm64"},
	}
}
