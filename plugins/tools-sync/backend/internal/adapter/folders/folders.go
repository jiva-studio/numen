// Package folders makes the folders on this computer that hold the vaults of received shares.
package folders

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	defaultFolder = "Numen Shared"
	maxNumber     = 1000
	folderMode    = 0o755
)

// ErrNoFreeFolder means every numbered folder of a name is taken.
var ErrNoFreeFolder = errors.New("no free folder for the received share")

// Received makes the folders of received shares in the folder of the setting, or in Numen Shared in the home folder
// of the person.
type Received struct {
	setting func() string
}

// NewReceived returns the folders in the folder setting returns.
func NewReceived(setting func() string) *Received { return &Received{setting: setting} }

// MakeReceivedFolder makes a folder of the name, with a number appended when the name is taken, and returns its path
// and the name of the vault it holds.
func (r *Received) MakeReceivedFolder(_ context.Context, name string) (ports.ReceivedFolder, error) {
	root, err := r.getRoot()
	if err != nil {
		return ports.ReceivedFolder{}, err
	}
	if err := os.MkdirAll(root, folderMode); err != nil {
		return ports.ReceivedFolder{}, fmt.Errorf("make %s: %w", root, err)
	}
	for number := 1; number <= maxNumber; number++ {
		numbered := name
		if number > 1 {
			numbered = fmt.Sprintf("%s %d", name, number)
		}
		folder := filepath.Join(root, numbered)
		err := os.Mkdir(folder, folderMode)
		if err == nil {
			return ports.ReceivedFolder{Path: folder, Name: numbered}, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return ports.ReceivedFolder{}, fmt.Errorf("make %s: %w", folder, err)
		}
	}
	return ports.ReceivedFolder{}, ErrNoFreeFolder
}

func (r *Received) getRoot() (string, error) {
	if configured := r.setting(); configured != "" {
		return configured, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find the home folder: %w", err)
	}
	return filepath.Join(home, defaultFolder), nil
}
