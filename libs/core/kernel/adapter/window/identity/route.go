package identity

import (
	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute returns the route IdentityService is served at, over the identity use cases on the key store,
// or over the handler that answers every call as unimplemented where the installation has no key store.
func NewRoute(
	keys pluginport.KeyStore, clock machineport.Clock, counted connect.HandlerOption,
) editor.Route {
	service := NewService(keys, clock)
	path, served := numenv1connect.NewIdentityServiceHandler(service, counted)
	return editor.NewRoute(path, served, service)
}

// NewService returns the handler over the identity use cases on the key store, or the handler that
// answers every call as unimplemented where the installation has no key store.
func NewService(keys pluginport.KeyStore, clock machineport.Clock) numenv1connect.IdentityServiceHandler {
	if keys == nil {
		return numenv1connect.UnimplementedIdentityServiceHandler{}
	}
	deps := identity.NewDeps(keys, hostName{}, clock)
	return NewHandler(UseCases{
		Get:           identity.NewGetIdentity(deps),
		GenerateWords: identity.NewGenerateRecoveryWords(deps),
		Create:        identity.NewCreateIdentity(deps),
		Delete:        identity.NewDeleteIdentity(deps),
	})
}
