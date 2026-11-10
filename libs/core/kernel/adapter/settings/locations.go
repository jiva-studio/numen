package settings

import "path/filepath"

// Adjacent returns the path of name in the folder of the registry file, and false while no registry path is
// set.
func Adjacent(registryPath, name string) (string, bool) {
	if registryPath == "" {
		return "", false
	}
	return filepath.Join(filepath.Dir(registryPath), name), true
}

// LocateFile returns the settings file: settingsPath where one is set, else numen.json beside the registry,
// else the file in the platform's own location.
func LocateFile(settingsPath, registryPath string) (string, error) {
	if settingsPath != "" {
		return settingsPath, nil
	}
	if file, isAdjacent := Adjacent(registryPath, "numen.json"); isAdjacent {
		return file, nil
	}
	return Path()
}

// LocateThemes returns the folder of the user's themes: themesPath where one is set, else the themes folder
// beside the registry, and false where neither is set and the platform's own folder holds them.
func LocateThemes(themesPath, registryPath string) (string, bool) {
	if themesPath != "" {
		return themesPath, true
	}
	return Adjacent(registryPath, "themes")
}
