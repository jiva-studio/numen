// Package hostclient reaches the vault, the text versions, the plugin's store and the work queue through
// the clients of the host.
package hostclient

import (
	"fmt"
	"os"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/usecase/texts"
)

// Host is the clients of the host the plugin works through.
type Host struct {
	files *host.Client
	texts *host.TextClient
	store *host.StoreClient
	queue host.Queue
}

// OpenFromEnv connects to the host the environment of the plugin process names.
func OpenFromEnv() (*Host, error) {
	textClient, err := host.NewTextClientFromEnv()
	if err != nil {
		return nil, err
	}
	store, err := host.NewStoreClientFromEnv()
	if err != nil {
		return nil, err
	}
	files, err := host.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	queue, err := host.NewQueueClientFromEnv()
	if err != nil {
		return nil, err
	}
	return &Host{files: files, texts: textClient, store: store, queue: queue}, nil
}

// OpenFromVars connects to the host the variables of a launched process of the plugin name.
func OpenFromVars(vars map[string]string) (*Host, error) {
	textClient, err := host.NewTextClientFromVars(vars)
	if err != nil {
		return nil, err
	}
	store, err := host.NewStoreClientFromVars(vars)
	if err != nil {
		return nil, err
	}
	files, err := host.NewClientFromVars(vars)
	if err != nil {
		return nil, err
	}
	queue, err := host.NewQueueClientFromVars(vars)
	if err != nil {
		return nil, err
	}
	return &Host{files: files, texts: textClient, store: store, queue: queue}, nil
}

// OpenFiles returns the files of a vault.
func (h *Host) OpenFiles(id domain.VaultID) host.FileReader {
	return h.files.With(id)
}

// OpenVault returns the files and the name of a vault.
func (h *Host) OpenVault(id domain.VaultID) texts.Vault {
	return h.files.With(id)
}

// OpenTexts returns the text versions of a vault.
func (h *Host) OpenTexts(id domain.VaultID) host.TextVersions {
	return h.texts.With(id)
}

// OpenStore returns the plugin's store in a vault.
func (h *Host) OpenStore(id domain.VaultID) host.StoreWriter {
	return h.store.With(id)
}

// GetQueue returns the work queue of the host.
func (h *Host) GetQueue() host.Queue {
	return h.queue
}

// ReportError writes a background failure of the plugin to standard error.
func (*Host) ReportError(what string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "readers-epub: %s: %v\n", what, err)
	}
}
