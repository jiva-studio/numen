package host_test

import (
	"net/http"

	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

func handler(a numenv1connect.HostVaultServiceHandler) http.Handler {
	mux := http.NewServeMux()
	path, h := numenv1connect.NewHostVaultServiceHandler(a)
	mux.Handle(path, h)
	return mux
}
