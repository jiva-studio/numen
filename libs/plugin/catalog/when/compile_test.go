package when

import (
	"errors"
	"testing"

	"cel.dev/cel-go/cel"
)

type badProgramLib struct{}

func (badProgramLib) CompileOptions() []cel.EnvOption { return nil }

func (badProgramLib) ProgramOptions() []cel.ProgramOption {
	return []cel.ProgramOption{cel.Globals(123)}
}

func TestConditionsParseErrors(t *testing.T) {
	wantErr := errors.New("env failed")
	brokenEnv := &Conditions{
		getEnvironment: func() (*cel.Env, error) { return nil, wantErr },
		compiled:       map[string]cacheEntry{},
	}
	if _, err := brokenEnv.Parse("target.isNote"); !errors.Is(err, wantErr) {
		t.Fatalf("brokenEnv.Parse err = %v, want %v", err, wantErr)
	}

	badProgramEnv := &Conditions{
		getEnvironment: func() (*cel.Env, error) {
			return cel.NewEnv(cel.Lib(badProgramLib{}))
		},
		compiled: map[string]cacheEntry{},
	}
	if _, err := badProgramEnv.Parse("true"); err == nil || errors.Unwrap(err) == nil {
		t.Fatalf("badProgramEnv.Parse err = %v, want wrapped program error", err)
	}
}
