package onnxruntime

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
)

// CacheDir is where a downloaded file is put: the folder the settings name, or
// the models folder every plugin shares.
func CacheDir(s Settings) (string, error) {
	if s.Dir != "" {
		return s.Dir, nil
	}
	return models.GetHome()
}

// Fetch is the file one address names, downloaded if it is not already here. A
// sum pins the bytes the address must hold; an empty sum fetches the file as
// named.
func Fetch(ctx context.Context, s Settings, address, sum string) (string, error) {
	dir, err := CacheDir(s)
	if err != nil {
		return "", err
	}
	return models.Fetch(ctx, models.Request{
		Dir:            dir,
		Section:        s.Section,
		Address:        address,
		Sum:            sum,
		ShouldDownload: s.ShouldDownload,
		Progress:       s.Fetching,
	})
}
