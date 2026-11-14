package environment

import (
	"strings"
	"testing"
)

func TestSessionVariablesAreDropped(t *testing.T) {
	got := DropSession([]string{"PATH=/bin", "CLAUDECODE=1", "CLAUDE_CODE_SESSION_ID=x", "HOME=/h"})
	if strings.Join(got, ",") != "PATH=/bin,HOME=/h" {
		t.Fatalf("got %v", got)
	}
}
