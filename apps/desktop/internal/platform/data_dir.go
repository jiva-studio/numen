package platform

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// indexFileName is the index database inside the application's data folder.
const indexFileName = "index.db"

// GetIndexPath is the index database in this machine's data folder for the application:
// `$XDG_DATA_HOME/numen` (default `~/.local/share/numen`) on Linux and other Unix systems,
// `~/Library/Application Support/numen` on macOS, `%AppData%\numen` on Windows.
func GetIndexPath() (string, error) {
	dir, err := getDataDir(runtime.GOOS, os.Getenv, os.UserHomeDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appFolder, indexFileName), nil
}

// appFolder is the application's folder inside the data folder.
const appFolder = "numen"

// getDataDir is the folder an operating system keeps user application data in.
func getDataDir(goos string, getEnv func(string) string, getHome func() (string, error)) (string, error) {
	switch goos {
	case "windows":
		if dir := getEnv("AppData"); dir != "" {
			return dir, nil
		}
		return "", errors.New("%AppData% is not set")
	case "darwin", "ios":
		home, err := getHome()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support"), nil
	}
	if dir := getEnv("XDG_DATA_HOME"); filepath.IsAbs(dir) {
		return dir, nil
	}
	home, err := getHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}
