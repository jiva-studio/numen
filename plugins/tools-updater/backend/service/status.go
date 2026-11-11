package service

import (
	"context"

	"github.com/jiva-studio/numen/plugins/tools-updater/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/release"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/updates"
)

func newStatusHandler(checker *updates.Checker) func(context.Context, api.GetStatusRequest) (api.Status, error) {
	return func(context.Context, api.GetStatusRequest) (api.Status, error) {
		return writeStatus(checker.GetStatus()), nil
	}
}

func writeStatus(status updates.Status) api.Status {
	return api.Status{State: api.State(status.State), Update: writeUpdate(status.Update)}
}

func writeUpdate(update *release.Update) *api.Update {
	if update == nil {
		return nil
	}
	var file *api.File
	if update.File != nil {
		file = &api.File{
			Name: update.File.Name, Arch: update.File.Arch, Kind: update.File.Kind,
			URL: update.File.URL, Size: update.File.Size, SHA256: update.File.SHA256,
		}
	}
	return &api.Update{
		Version: update.Version, Build: int64(update.Build), Channel: update.Channel,
		ReleasedAt: update.ReleasedAt, NotesURL: update.NotesURL, DownloadURL: update.DownloadURL,
		File: file,
	}
}
