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
	Value string `json:"value"`
	// Title is what the person reads.
	Title string `json:"title"`
	// Group is the heading the option is shown under.
	Group string `json:"group"`
}

// OptionsQuery is what the core asks a list of options with.
//
//numen:api
type OptionsQuery struct {
	// Operations are the operations the setting of rows the field belongs to is used for.
	Operations []string
	// IsFresh asks for the options read anew, past any list the provider keeps.
	IsFresh bool
}

// ListedOptions is the answer of a list of options.
//
//numen:api
type ListedOptions struct {
	Options []Option
	// IsPartial marks a list that misses values the provider could not read now. The core does not keep it,
	// and asks again the next time.
	IsPartial bool
}

// ListOptions lists the values of a row field for the connection of call.
type ListOptions func(ctx context.Context, call Call, query OptionsQuery) (ListedOptions, error)

// ServeOptions is the operation a row field of a connection type names in optionsFrom: it reads the query
// from the payload and answers the options as the done payload. The handler of the provider releases the
// process after the answer where no other call is open.
func ServeOptions(list ListOptions) Operation {
	return func(ctx context.Context, call Call, payload []byte, _ func([]byte) error) ([]byte, error) {
		var request v1.OptionsRequest
		if err := proto.Unmarshal(payload, &request); err != nil {
			return nil, NewStopError(ReasonInvalidRequest)
		}
		listed, err := list(ctx, call, OptionsQuery{Operations: request.GetOperations(), IsFresh: request.GetIsFresh()})
		if err != nil {
			return nil, err
		}
		markLookup(ctx)
		wire := &v1.Options{Options: make([]*v1.Option, len(listed.Options)), IsPartial: listed.IsPartial}
		for i, one := range listed.Options {
			wire.Options[i] = &v1.Option{Value: one.Value, Title: one.Title, Group: one.Group}
		}
		return proto.Marshal(wire)
	}
}
