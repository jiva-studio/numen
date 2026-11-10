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

func TestAMapBecomesNameValueEntries(t *testing.T) {
	vars := FromMap(map[string]string{"K": "V"})
	if len(vars) != 1 || vars[0] != "K=V" {
		t.Errorf("FromMap = %v, want [K=V]", vars)
	}
}
