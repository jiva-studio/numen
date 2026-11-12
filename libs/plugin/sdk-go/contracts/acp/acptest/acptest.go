// Package acptest carries out the acp.task operation a plugin provides for the tests of the plugin, with
// the messages of the contract in place of their encoding.
package acptest

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
)

// RunTask asks the acp.task operation of provider to carry out the prompt turn of request. It hands each
// update to send, in order, and returns the stop reason.
func RunTask(
	ctx context.Context, provider connection.Provider, call connection.Call, request *acp.TaskRequest, send func(*acp.TaskFrame) error,
) (*acp.TaskDone, error) {
	served, isServed := provider[acp.OperationTask]
	if !isServed {
		return nil, fmt.Errorf("acptest: the provider does not serve %s", acp.OperationTask)
	}
	payload, err := EncodeTaskRequest(request)
	if err != nil {
		return nil, err
	}
	done, err := served(ctx, call, payload, func(data []byte) error {
		frame := &acp.TaskFrame{}
		if err := proto.Unmarshal(data, frame); err != nil {
			return err
		}
		return send(frame)
	})
	if err != nil {
		return nil, err
	}
	reason := &acp.TaskDone{}
	if err := proto.Unmarshal(done, reason); err != nil {
		return nil, err
	}
	return reason, nil
}

// EncodeTaskRequest returns the payload of an acp.task call for request, as a test sends it over the wire.
func EncodeTaskRequest(request *acp.TaskRequest) ([]byte, error) {
	return proto.Marshal(request)
}
