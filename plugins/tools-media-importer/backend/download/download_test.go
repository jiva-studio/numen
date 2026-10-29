package download

import (
	"bytes"
	"errors"
	"os/exec"
	"testing"
)

// A program run carries the environment it is given.
func TestProgramBuildCommandSetsEnvironment(t *testing.T) {
	out, err := run(t.Context(), program{
		command: []string{"sh", "-c", `printf '%s' "$NUMEN_TEST_ENV"`},
		env:     []string{"NUMEN_TEST_ENV=passed_value"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "passed_value" {
		t.Errorf("run with environment returned %q, want %q", string(out), "passed_value")
	}
}

// A run directs standard output to the given writer when provided.
func TestRunRoutesOutputToCustomWriter(t *testing.T) {
	var into bytes.Buffer
	out, err := run(t.Context(), program{command: []string{"sh", "-c", `printf 'streamed'`}}, &into)
	if err != nil {
		t.Fatal(err)
	}
	if into.String() != "streamed" {
		t.Errorf("into buffer received %q, want %q", into.String(), "streamed")
	}
	if len(out) != 0 {
		t.Errorf("run output returned %q, want empty bytes when into is provided", string(out))
	}
}

// A failed run wraps the process exit error.
func TestRunWrapsExecutionError(t *testing.T) {
	_, err := run(t.Context(), program{command: []string{"sh", "-c", `echo failed >&2; exit 2`}}, nil)
	if err == nil {
		t.Fatal("run did not fail")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Errorf("error %v does not wrap *exec.ExitError", err)
	}
}

// A program that produces no output reports that it printed nothing.
func TestGetLastLine_ReturnsFallbackWhenEmpty(t *testing.T) {
	if got := getLastLine(""); got != "it said nothing" {
		t.Errorf("getLastLine(\"\") = %q, want %q", got, "it said nothing")
	}
	if got := getLastLine("   \n\t  \n  "); got != "it said nothing" {
		t.Errorf("getLastLine(whitespace) = %q, want %q", got, "it said nothing")
	}
	if got := getLastLine("first\nsecond\n"); got != "second" {
		t.Errorf("getLastLine(multiline) = %q, want %q", got, "second")
	}
}

func TestResolveProgramPrecedence(t *testing.T) {
	if got := resolveProgram(Tool{Command: []string{"/setting/tool"}}, "sh").command[0]; got != "/setting/tool" {
		t.Errorf("setting path = %q, want /setting/tool", got)
	}
	if got := resolveProgram(Tool{}, "sh"); !got.isPresent() || got.command[0] == "sh" {
		t.Errorf("PATH lookup = %v, want the sh found on PATH", got.command)
	}
	if got := resolveProgram(Tool{}, "no-such-numen-tool"); got.isPresent() {
		t.Errorf("missing tool resolved to %v", got.command)
	}
}
