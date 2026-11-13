package connection

import (
	"context"

	"google.golang.org/protobuf/proto"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

// OperationDetected is the operation through which a provider reports the connections of a connection type it
// found ready on this machine. A type takes part by listing it among its operations in the manifest; the core
// asks when the plugin starts and when the person opens the list of connections, and lists what the provider
// answered after the stored connections.
const OperationDetected = "connections.detected"

// DetectedConnection is a connection a provider found ready on this machine.
//
//numen:api
type DetectedConnection struct {
	// Name is what the person reads, unique within the type. It is part of the id of the connection.
	Name string `json:"name"`
	// Fields are the values of fields of the type that are not secret, by key; a field left out takes its
	// default.
	Fields map[string]string `json:"fields"`
}

// DetectConnections returns the connections of the type of call found on this machine, none when nothing is
// found. call names the type and no connection.
type DetectConnections func(ctx context.Context, call Call) ([]DetectedConnection, error)

// ServeDetected is the operation connections.detected: it answers the connections detect finds as the done
// payload.
func ServeDetected(detect DetectConnections) Operation {
	return func(ctx context.Context, call Call, _ []byte, _ func([]byte) error) ([]byte, error) {
		found, err := detect(ctx, call)
		if err != nil {
			return nil, err
		}
		wire := &v1.DetectedConnections{Connections: make([]*v1.DetectedConnection, len(found))}
		for i, one := range found {
			wire.Connections[i] = &v1.DetectedConnection{Name: one.Name, Fields: one.Fields}
		}
		return proto.Marshal(wire)
	}
}
