// Package host reaches the notes of the vault through the host the plugin process was launched
// for. It offers the calls of the index and the write of a ticked task, and no other.
package host

import (
	"context"

	sdkhost "github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// Vault is the notes of the vault the host shows.
type Vault struct {
	client *sdkhost.Client
}

// FromEnv connects to the host the environment of the plugin process names.
func FromEnv() (*Vault, error) {
	client, err := sdkhost.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	return &Vault{client: client}, nil
}

func (v *Vault) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	return v.client.StatFile(ctx, path)
}

func (v *Vault) Read(ctx context.Context, path string) ([]byte, domain.FileStat, error) {
	return v.client.Read(ctx, path)
}

func (v *Vault) GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error) {
	return v.client.GetPathsOfType(ctx, noteType)
}

func (v *Vault) Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error) {
	return v.client.Write(ctx, path, content, baseline)
}
