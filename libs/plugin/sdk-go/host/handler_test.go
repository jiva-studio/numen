package host_test

import (
	"net/http"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// unimplementedHost answers every procedure of the host vault services with unimplemented; a fake host
// embeds it and answers the procedures its test calls.
type unimplementedHost struct {
	numenv1connect.UnimplementedHostVaultServiceHandler
	numenv1connect.UnimplementedHostFileServiceHandler
	numenv1connect.UnimplementedHostNoteServiceHandler
	numenv1connect.UnimplementedHostLinkServiceHandler
	numenv1connect.UnimplementedHostCacheServiceHandler
	numenv1connect.UnimplementedVaultsServiceHandler
}

type hostServices interface {
	numenv1connect.HostVaultServiceHandler
	numenv1connect.HostFileServiceHandler
	numenv1connect.HostNoteServiceHandler
	numenv1connect.HostLinkServiceHandler
	numenv1connect.HostCacheServiceHandler
	numenv1connect.VaultsServiceHandler
}

func handler(a hostServices) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostVaultServiceHandler(a))
	mux.Handle(numenv1connect.NewHostFileServiceHandler(a))
	mux.Handle(numenv1connect.NewHostNoteServiceHandler(a))
	mux.Handle(numenv1connect.NewHostLinkServiceHandler(a))
	mux.Handle(numenv1connect.NewHostCacheServiceHandler(a))
	mux.Handle(numenv1connect.NewVaultsServiceHandler(a))
	return mux
}

func vaultsErrorOf(code v1.VaultsErrorCode) *v1.VaultsErrorCode { return &code }
