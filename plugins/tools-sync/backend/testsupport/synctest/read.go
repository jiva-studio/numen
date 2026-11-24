package synctest

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/catalog"
)

const (
	// LinkFile, FilesFile and FeedFile are the files of the store folder of the plugin in each vault.
	LinkFile  = "link.json"
	FilesFile = "files.json"
	FeedFile  = "feed.json"

	stateKeyContext = "numen-sync/state-key/v1"
	catalogContext  = "numen-sync/catalog/v1"
)

// FileRow is a row of files.json as the tests read it.
type FileRow struct {
	FileID   json.RawMessage `json:"fileId"`
	Path     string          `json:"path"`
	BaseHash string          `json:"baseHash"`
	HeadRev  json.RawMessage `json:"headRev"`
}

// Connect connects the vault as a new synced vault.
func (c *Computer) Connect(t *testing.T, id domain.VaultID) {
	t.Helper()
	in := catalog.ConnectInput{VaultID: id, Mode: catalog.ModeNew, RemoteID: ""}
	if err := catalog.NewConnectVault(c.CatalogDeps()).Execute(t.Context(), in); err != nil {
		t.Fatal(err)
	}
}

func (c *Computer) ConnectExisting(t *testing.T, id domain.VaultID, remoteID string) {
	t.Helper()
	in := catalog.ConnectInput{VaultID: id, Mode: catalog.ModeExisting, RemoteID: remoteID}
	if err := catalog.NewConnectVault(c.CatalogDeps()).Execute(t.Context(), in); err != nil {
		t.Fatal(err)
	}
}

// ReadStoreFile returns a file of the store folder of the plugin in the vault.
func (c *Computer) ReadStoreFile(t *testing.T, id domain.VaultID, name string) []byte {
	t.Helper()
	raw, err := c.Host.Store(id).Read(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// Link returns link.json of the vault.
func (c *Computer) Link(t *testing.T, id domain.VaultID) vaultlink.Link {
	t.Helper()
	var link vaultlink.Link
	if err := json.Unmarshal(c.ReadStoreFile(t, id, LinkFile), &link); err != nil {
		t.Fatal(err)
	}
	return link
}

// GetRemoteID returns the id the server knows the vault by.
func (c *Computer) GetRemoteID(t *testing.T, id domain.VaultID) []byte {
	t.Helper()
	remote, err := hex.DecodeString(c.Link(t, id).RemoteVaultID)
	if err != nil {
		t.Fatal(err)
	}
	return remote
}

// GetVaultKey opens the key of the vault from link.json.
func (c *Computer) GetVaultKey(t *testing.T, id domain.VaultID) [32]byte {
	t.Helper()
	remote := c.GetRemoteID(t, id)
	opened, err := c.Client.OpenData(t.Context(), append([]byte(stateKeyContext), remote...), c.Link(t, id).SealedKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(opened) != 32 {
		t.Fatalf("the vault key has %d bytes, want 32", len(opened))
	}
	return [32]byte(opened)
}

func (c *Computer) ReadFileRows(t *testing.T, id domain.VaultID) []FileRow {
	t.Helper()
	var parsed struct {
		Files []FileRow `json:"files"`
	}
	if err := json.Unmarshal(c.ReadStoreFile(t, id, FilesFile), &parsed); err != nil {
		t.Fatal(err)
	}
	return parsed.Files
}

func (c *Computer) GetFileRow(t *testing.T, id domain.VaultID, path string) (FileRow, bool) {
	t.Helper()
	rows := c.ReadFileRows(t, id)
	at := slices.IndexFunc(rows, func(row FileRow) bool { return row.Path == path })
	if at < 0 {
		return FileRow{}, false
	}
	return rows[at], true
}

// ReadCatalog returns every entry of the catalog of the account on the server.
func (c *Computer) ReadCatalog(t *testing.T) []*syncv1.CatalogEntry {
	t.Helper()
	entries, err := c.Cloud.GetCatalog(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

// ReadChanges returns the head revision of every file of the vault on the server.
func (c *Computer) ReadChanges(t *testing.T, remoteID []byte) []*syncv1.StoredRevision {
	t.Helper()
	var all []*syncv1.StoredRevision
	var since uint64
	for {
		page, err := c.Cloud.GetChanges(t.Context(), remoteID, since, 0)
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, page.GetChanges()...)
		if !page.GetHasMore() {
			return all
		}
		since = page.GetNextSeq()
	}
}

func (c *Computer) OpenCatalogBody(t *testing.T, entry *syncv1.CatalogEntry) CatalogBody {
	t.Helper()
	opened, err := c.Client.OpenData(t.Context(), GetCatalogContext(entry), entry.GetEncBody())
	if err != nil {
		t.Fatal(err)
	}
	var body CatalogBody
	if err := json.Unmarshal(opened, &body); err != nil {
		t.Fatal(err)
	}
	return body
}

// CatalogBody is the body of a catalog entry once opened.
type CatalogBody struct {
	LocalID string `json:"localId"`
	Name    string `json:"name"`
	Key     []byte `json:"key"`
}

// WriteNote writes a file in the vault as another window would, and the host indexes it.
func (c *Computer) WriteNote(id domain.VaultID, path, text string) {
	c.Vaults.With(id).WriteFile(path, text)
}

// ReadNote returns the text of the file in the vault, and whether the file is there.
func (c *Computer) ReadNote(id domain.VaultID, path string) (string, bool) {
	return c.Vaults.With(id).ReadContent(path)
}

// HasItem reports whether the health host holds an item of the text.
func (c *Computer) HasItem(text string) bool {
	return slices.ContainsFunc(c.Health.List(), func(item health.Item) bool { return item.Text == text })
}

// HasItemKey reports whether the health host holds an item of the key.
func (c *Computer) HasItemKey(key string) bool {
	return slices.ContainsFunc(c.Health.List(), func(item health.Item) bool { return item.Key == key })
}

// GetCatalogContext returns the context the body of the catalog entry is sealed under.
func GetCatalogContext(entry *syncv1.CatalogEntry) []byte {
	sealedFor := append([]byte(catalogContext), entry.GetVaultId()...)
	return binary.BigEndian.AppendUint64(sealedFor, entry.GetVersion())
}
