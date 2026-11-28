//go:build numen_debug

package platform

import (
	"errors"
	"io/fs"
	"net"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/apps/desktop/internal/keychain"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/identityfile"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

// ApplyDebugIdentityFiles selects the credentials and chain files of an isolated loopback profile with plaintext storage.
func ApplyDebugIdentityFiles(address string, res *container.Resources) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil || !net.ParseIP(host).IsLoopback() {
		return false
	}
	folder := getDebugIdentityFolder()
	if folder == "" {
		return false
	}
	store := keychain.NewDebugFileStore(folder)
	container.SetKeys(res, store)
	res.KeyFile = keychain.NewKeyFile(store, &keychain.Passphrase{})
	res.Stores.Chains = identityfile.Open(folder)
	return true
}

func getDebugIdentityFolder() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	home, err = filepath.EvalSymlinks(home)
	if err != nil {
		return ""
	}
	profile := filepath.Dir(home)
	temp, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil || !isWithinDebugProfile(temp, profile) {
		return ""
	}
	path, err := settings.GetPath()
	if err != nil {
		return ""
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil || !isWithinDebugProfile(profile, path) {
		return ""
	}
	if settings.ReadCredentialsFile(path) != settings.CredentialsFilePlaintext {
		return ""
	}
	folder := filepath.Dir(path)
	if !hasDebugIdentityPaths(folder) {
		return ""
	}
	return folder
}

func isWithinDebugProfile(profile, path string) bool {
	relative, err := filepath.Rel(profile, path)
	return err == nil && relative != "." && filepath.IsLocal(relative)
}

func hasDebugIdentityPaths(folder string) bool {
	for _, path := range []string{
		filepath.Join(folder, "credentials", "credentials.json"),
		filepath.Join(folder, "identity", "chain.bin"),
	} {
		if !isDebugProfilePath(folder, path) {
			return false
		}
	}
	return true
}

func isDebugProfilePath(folder, path string) bool {
	for path != folder {
		_, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			path = filepath.Dir(path)
			continue
		}
		if err != nil {
			return false
		}
		resolved, err := filepath.EvalSymlinks(path)
		return err == nil && resolved == path
	}
	return true
}
