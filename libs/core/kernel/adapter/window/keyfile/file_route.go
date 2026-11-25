package keyfile

import (
	"net/http"

	"connectrpc.com/connect"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	keyfileusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/keyfile"
)

// NewFileRoute creates the route of the key file service over the key file. Each call goes through counted.
func NewFileRoute(file keyfileusecase.File, counted connect.HandlerOption) (string, http.Handler) {
	return NewRoute(NewHandler(UseCases{
		GetState: keyfileusecase.NewGetState(file),
		Choose:   keyfileusecase.NewChoose(file, settingsadapter.WriteKeyFileMode),
		Unlock:   keyfileusecase.NewUnlock(file),
	}), counted)
}
