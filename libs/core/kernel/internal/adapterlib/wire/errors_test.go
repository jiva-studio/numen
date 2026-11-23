package wire_test

import (
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"testing"

	"connectrpc.com/connect"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
)

func TestGetErrorDetailIsTheReasonAfterTheSentinelAndNothingForAnotherError(t *testing.T) {
	sentinel := errors.New("name taken")
	if got := wire.GetErrorDetail(fmt.Errorf("%w: Ollama", sentinel), sentinel); got != "Ollama" {
		t.Errorf("detail = %q, want Ollama", got)
	}
	if got := wire.GetErrorDetail(errors.New("open /home/user/key: denied"), sentinel); got != "" {
		t.Errorf("an error that wraps no sentinel gave detail %q", got)
	}
}

func TestNewListErrorNamesWhoseFaultAPathIs(t *testing.T) {
	cases := map[string]struct {
		err  error
		want connect.Code
	}{
		"outside the vault": {vaultdomain.ErrOutside, connect.CodeInvalidArgument},
		"nothing there":     {fs.ErrNotExist, connect.CodeNotFound},
		"anything else":     {errors.New("disk"), connect.CodeInternal},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			if got := connect.CodeOf(wire.NewListError(one.err)); got != one.want {
				t.Errorf("code = %v, want %v", got, one.want)
			}
		})
	}
}

func TestEachOnceDropsRepeatsAndRefusesTooManyPaths(t *testing.T) {
	got, err := wire.EachOnce([]string{"a", "b", "a"})
	if err != nil || len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("paths = %v, err = %v", got, err)
	}
	many := make([]string, 201)
	for i := range many {
		many[i] = strconv.Itoa(i)
	}
	if _, err := wire.EachOnce(many); connect.CodeOf(err) != connect.CodeInvalidArgument || !errors.Is(err, wire.ErrTooManyPaths) {
		t.Errorf("err = %v", err)
	}
}
