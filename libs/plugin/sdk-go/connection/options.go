package connection

import (
	"context"

	"google.golang.org/protobuf/proto"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

// Option is one value a row field can take. An empty Title shows the value, and an empty Group shows no
// heading.
//
//numen:api
type Option struct {
	// Value is what the row stores.
	Value string
	// Title is what the person reads.
	Title string
	// Group is the heading the option is shown under.
	Group string
}

// ListOptions lists the values of a row field for the connection of call. operations are the operations
// the setting of rows the field belongs to is used for.
type ListOptions func(ctx context.Context, call Call, operations []string) ([]Option, error)

// ServeOptions is the operation a row field of a connection type names in optionsFrom: it reads the
// operations of the setting from the payload and answers the options as the done payload.
func ServeOptions(list ListOptions) Operation {
	return func(ctx context.Context, call Call, payload []byte, _ func([]byte) error) ([]byte, error) {
		var request v1.OptionsRequest
		if err := proto.Unmarshal(payload, &request); err != nil {
			return nil, NewStopError(ReasonInvalidRequest)
		}
		listed, err := list(ctx, call, request.GetOperations())
		if err != nil {
			return nil, err
		}
		wire := &v1.Options{Options: make([]*v1.Option, len(listed))}
		for i, one := range listed {
			wire.Options[i] = &v1.Option{Value: one.Value, Title: one.Title, Group: one.Group}
		}
		return proto.Marshal(wire)
	}
}
