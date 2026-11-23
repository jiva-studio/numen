package fakeserver

import "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"

// shareService serves the share service with the rules of the numen-cloud server.
type shareService struct {
	syncv1connect.UnimplementedShareServiceHandler
}
