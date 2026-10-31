package onnx

import (
	"fmt"
	"os"
	"path/filepath"
)

// ModelsDirVar names the environment variable that holds the folder plugins keep their models in.
const ModelsDirVar = "NUMEN_MODELS_DIR"

// FindInFolder returns the path of fileName in dir, or "" where dir or fileName is empty or the
// file is absent.
func FindInFolder(dir, fileName string) string {
	if dir == "" || fileName == "" {
		return ""
	}
	candidate := filepath.Join(dir, fileName)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

// FindInHome returns the path of fileName in the models folder the environment names, else in the
// models folder of the user configuration, or "" where neither holds it.
func FindInHome(fileName string) string {
	if home := os.Getenv(ModelsDirVar); home != "" {
		if found := FindInFolder(home, fileName); found != "" {
			return found
		}
	}
	if base, err := os.UserConfigDir(); err == nil {
		return FindInFolder(filepath.Join(base, "numen", "models"), fileName)
	}
	return ""
}

// ResolveConfigured returns the model the configuration names: explicitPath, which must exist, else
// the file the environment variable envVar names when it exists. It returns "" where the
// configuration names none.
func ResolveConfigured(explicitPath, envVar string) (string, error) {
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err != nil {
			return "", fmt.Errorf("configured model path not found: %w", err)
		}
		return explicitPath, nil
	}
	if envVar != "" {
		if env := os.Getenv(envVar); env != "" {
			if _, err := os.Stat(env); err == nil {
				return env, nil
			}
		}
	}
	return "", nil
}

// LocateInFolders returns the path of fileName in dir, else in the models folders of FindInHome.
func LocateInFolders(dir, fileName string) (string, error) {
	if found := FindInFolder(dir, fileName); found != "" {
		return found, nil
	}
	if found := FindInHome(fileName); found != "" {
		return found, nil
	}
	return "", fmt.Errorf("model %s not found", fileName)
}

// LocateModel finds a model file by checking, in order, the explicit path, the environment variable
// and the folders of LocateInFolders.
func LocateModel(explicitPath, envVar, fileName, dir string) (string, error) {
	configured, err := ResolveConfigured(explicitPath, envVar)
	if err != nil || configured != "" {
		return configured, err
	}
	return LocateInFolders(dir, fileName)
}
