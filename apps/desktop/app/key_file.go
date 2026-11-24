package app

import (
	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	windowkeyfile "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/keyfile"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/keyfile"
)

// mountKeyFile serves KeyFileService to the window: this host owns the key file, not the plugin host.
func mountKeyFile(installation *editor.Installation, file pluginport.KeyFile) {
	handler := windowkeyfile.NewHandler(windowkeyfile.UseCases{
		GetState: keyfile.NewGetState(file),
		Choose:   keyfile.NewChoose(file, rememberKeyFile),
		Unlock:   keyfile.NewUnlock(file),
	})
	path, served := windowkeyfile.NewRoute(handler, connect.WithInterceptors())
	installation.Mount(editor.NewRoute(path, served))
}

func rememberKeyFile(mode pluginport.KeyFileMode) error {
	path, err := settings.GetPath()
	if err != nil {
		return err
	}
	return settings.WriteCredentialsFile(path, string(mode))
}
