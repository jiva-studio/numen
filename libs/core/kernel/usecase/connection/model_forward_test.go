package connection_test

import (
	"errors"
	"testing"

	connectiondomain "github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	modelcalldomain "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection/modelcalltest"
)

func TestForwardedCallsReachTheCallsSetLaterAndFindNoConnectionBefore(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	s.world.Provider.SetScript(addressOne, nil, "partial")
	var current connectionport.ModelCalls
	forwarded := connection.NewForwardedCalls(func() connectionport.ModelCalls { return current })
	request := modelcalldomain.Request{Operation: modelcalltest.ChatOperation}

	_, before := forwarded.Call(t.Context(), consumerID, request, func([]byte) error { return nil })
	current = s.calls
	var frames string
	done, after := forwarded.Call(t.Context(), consumerID, request, func(data []byte) error {
		frames += string(data)
		return nil
	})

	if !errors.Is(before, connectiondomain.ErrNoConnection) {
		t.Errorf("before the calls are set: err = %v, want no connection", before)
	}
	if after != nil || frames != "partial" || string(done) != "answer from "+addressOne {
		t.Errorf("after the calls are set: %q, %q, %v", frames, done, after)
	}
}
