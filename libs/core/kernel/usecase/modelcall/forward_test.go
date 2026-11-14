package modelcall_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	modelcalldomain "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall/modelcalltest"
)

func TestForwardedCallsReachTheCallsSetLaterAndFindNoConnectionBefore(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	s.world.Provider.Script(addressOne, nil, "partial")
	var current connectionport.ModelCalls
	forwarded := modelcall.NewForwardedCalls(func() connectionport.ModelCalls { return current })
	request := modelcalldomain.Request{Operation: modelcalltest.ChatOperation}

	_, before := forwarded.Call(t.Context(), consumerID, request, func([]byte) error { return nil })
	current = s.calls
	var frames string
	done, after := forwarded.Call(t.Context(), consumerID, request, func(data []byte) error {
		frames += string(data)
		return nil
	})

	if !errors.Is(before, connection.ErrNoConnection) {
		t.Errorf("before the calls are set: err = %v, want no connection", before)
	}
	if after != nil || frames != "partial" || string(done) != "answer from "+addressOne {
		t.Errorf("after the calls are set: %q, %q, %v", frames, done, after)
	}
}
