package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/notepath"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/arrange"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/files"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/library"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/links"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/notes"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/vaults"
)

// call holds what one tool call reads of the vault it names.
type call struct {
	tools   *Tools
	current domain.VaultID
	vault   Vault
	texts   Library
}

func (t *Tools) open(ctx context.Context) (*call, error) {
	id := tools.GetVault(ctx)
	if id == "" {
		return nil, errNoVault
	}
	c := &call{tools: t, current: domain.VaultID(id), vault: t.vaults(domain.VaultID(id)), texts: Library{Versions: nil, Texts: nil, Search: nil}}
	if t.library != nil {
		c.texts = t.library(c.current)
	}
	return c, nil
}

func (c *call) createNotes() *notes.UseCase {
	return notes.New(c.vault.Files, c.vault.Writer, c.vault.Index, c.vault.Notes)
}

func (c *call) createFiles() *files.UseCase {
	return files.New(c.vault.Files, c.vault.Writer, c.vault.Index)
}

func (c *call) createLinks() *links.UseCase {
	return links.New(c.vault.Links, c.vault.Editor, notepath.NewResolver(c.vault.Notes, c.vault.Files))
}

func (c *call) createArrange() *arrange.UseCase {
	return arrange.New(c.vault.Arranger)
}

func (c *call) createLibrary() *library.UseCase {
	return library.New(c.vault.Files, c.vault.Notes, c.texts.Versions, c.texts.Texts, c.texts.Search)
}

func (c *call) createVaults() *vaults.UseCase {
	open := func(id domain.VaultID) vaults.Vault {
		vault := c.tools.vaults(id)
		return vaults.Vault{Selector: vault.Selector, Notes: vault.Notes}
	}
	return vaults.New(c.current, c.tools.list, open)
}
