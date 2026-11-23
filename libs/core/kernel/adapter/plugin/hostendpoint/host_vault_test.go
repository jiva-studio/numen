package hostendpoint

import (
	"context"
	"errors"
	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// memoryVault replaces the notes and links of a vault behind the endpoint, which
// the real use cases serve from an index.
type memoryVault struct {
	mu        sync.Mutex
	vault     *domain.Vault
	files     map[string]string
	stamp     int64
	others    map[domain.VaultID]*memoryVault
	storeRoot string
}

func newMemoryVault(files map[string]string) *memoryVault {
	return &memoryVault{
		vault: &domain.Vault{ID: "v1", Name: "Physics", Path: "/vaults/Physics"},
		files: files,
	}
}

func (m *memoryVault) GetVaults() ([]domain.Vault, error) {
	out := []domain.Vault{}
	if m.vault != nil {
		out = append(out, *m.vault)
	}
	for _, other := range m.others {
		out = append(out, *other.vault)
	}
	slices.SortFunc(out, func(a, b domain.Vault) int { return strings.Compare(string(a.ID), string(b.ID)) })
	return out, nil
}

func (m *memoryVault) With(id domain.VaultID) (*vaultapi.Access, error) {
	if m.vault != nil && m.vault.ID == id {
		return vaultOf(m), nil
	}
	if other, there := m.others[id]; there {
		return vaultOf(other), nil
	}
	return nil, fs.ErrNotExist
}

func (m *memoryVault) GetVault() (domain.Vault, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.vault == nil {
		return domain.Vault{}, false
	}
	return *m.vault, true
}

func (m *memoryVault) stat(path string) (domain.FileStat, error) {
	body, there := m.files[path]
	if !there {
		return domain.FileStat{}, fs.ErrNotExist
	}
	return domain.FileStat{Path: path, Size: int64(len(body)), ModTime: time.Unix(0, 100+m.stamp)}, nil
}

func (m *memoryVault) StatFile(_ context.Context, path string) (domain.FileStat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stat(path)
}

func (m *memoryVault) Read(_ context.Context, path string) ([]byte, domain.FileStat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pos, err := m.stat(path)
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	return []byte(m.files[path]), pos, nil
}

func (m *memoryVault) ReadRange(context.Context, string, int, int) (plugin.FileRange, error) {
	return plugin.FileRange{}, errors.New("not implemented")
}

func (m *memoryVault) AddLinks(context.Context, []plugin.LinkAddition) ([]plugin.LinkAdded, error) {
	return nil, errors.New("not implemented")
}

func (m *memoryVault) UpdateLinks(context.Context, string, domain.Address, domain.Link, domain.FileStat) (domain.FileStat, error) {
	return domain.FileStat{}, errors.New("not implemented")
}

func (m *memoryVault) RemoveLinks(context.Context, string, domain.Address, domain.LinkRole, domain.FileStat) (domain.FileStat, error) {
	return domain.FileStat{}, errors.New("not implemented")
}

func (m *memoryVault) WriteFile(_ context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now, err := m.stat(path)
	if err != nil {
		return domain.FileStat{}, err
	}
	if !now.ModTime.Equal(baseline.ModTime) {
		return domain.FileStat{}, edit.ErrStale
	}
	m.stamp++
	m.files[path] = string(content)
	return m.stat(path)
}

func (m *memoryVault) CreateFile(_ context.Context, path string, content []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, there := m.files[path]; there {
		return vaultdomain.ErrAlreadyExists
	}
	m.files[path] = string(content)
	return nil
}

func (m *memoryVault) GetNotes(_ context.Context, paths []string) (map[string]domain.NoteRef, error) {
	out := map[string]domain.NoteRef{}
	for _, path := range paths {
		if _, there := m.files[path]; there {
			out[path] = domain.NoteRef{Path: path, Title: "Title of " + path}
		}
	}
	return out, nil
}

func (m *memoryVault) GetTypes(_ context.Context, paths []string) (map[string]domain.NoteType, error) {
	out := map[string]domain.NoteType{}
	for _, path := range paths {
		if _, there := m.files[path]; there {
			out[path] = "sample"
		}
	}
	return out, nil
}

func (m *memoryVault) ListNamedPaths(_ context.Context, name string) ([]string, error) {
	var out []string
	for path := range m.files {
		if domain.GetBasename(path) == name {
			out = append(out, path)
		}
	}
	slices.Sort(out)
	return out, nil
}

func (m *memoryVault) ListPathsOfType(_ context.Context, noteType domain.NoteType) ([]string, error) {
	if noteType != "sample" {
		return nil, nil
	}
	paths := make([]string, 0, len(m.files))
	for path := range m.files {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return paths, nil
}

func (m *memoryVault) link(from string) domain.ResolvedLink {
	return domain.ResolvedLink{
		Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Animal"}, Type: "preset", Label: "by"},
		From: from, To: "Animal.md", ToVault: "v1", IsAmbiguous: true,
	}
}

func (m *memoryVault) ListLinks(_ context.Context, from string) ([]domain.ResolvedLink, error) {
	return []domain.ResolvedLink{m.link(from)}, nil
}

func (m *memoryVault) ListBacklinks(_ context.Context, to string) ([]domain.ResolvedLink, error) {
	back := m.link("samples/" + to)
	back.Role = domain.RoleParent
	back.Why = "because"
	back.Marker = domain.MarkerArrow
	return []domain.ResolvedLink{back}, nil
}

func (m *memoryVault) ResolveLinks(_ context.Context, from string, output []string) (map[string]domain.ResolvedLink, error) {
	out := map[string]domain.ResolvedLink{}
	for _, one := range output {
		out[one] = m.link(from)
	}
	return out, nil
}

func (m *memoryVault) ListParseProblems(context.Context) ([]domain.VaultProblem, error) {
	return []domain.VaultProblem{{Path: "samples/A.md", Detail: "an entry has no role"}}, nil
}

func (m *memoryVault) RenameNote(context.Context, string, string) (string, bool, *domain.MoveResult, error) {
	return "", false, nil, errors.ErrUnsupported
}

func (m *memoryVault) MoveNote(context.Context, string, string) (domain.MoveResult, error) {
	return domain.MoveResult{}, errors.ErrUnsupported
}

func (m *memoryVault) RemoveNote(context.Context, string) (string, []string, error) {
	return "", nil, errors.ErrUnsupported
}

func (m *memoryVault) GetProblems(context.Context, []domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
	return nil, nil, errors.ErrUnsupported
}

func newVaultClient(t *testing.T, fakeCallers *callers, token string) HostVaultClient {
	t.Helper()
	endpoint := &Endpoint{callers: fakeCallers.asCallers()}
	mux := http.NewServeMux()
	vaultapi.Mount(mux, endpoint, endpoint.callers.Lent)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return NewHostVaultClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor(token)))
}

func newVaultFixture(t *testing.T) (HostVaultClient, *memoryVault) {
	t.Helper()
	vault := newMemoryVault(map[string]string{"samples/A.md": "body", "Animal.md": "form"})
	hosted := newCallers()
	hosted.vault = vaultOf(vault)
	return newVaultClient(t, hosted, "token-a"), vault
}

func TestHostVault_NoVaultReturnsEmptyList(t *testing.T) {
	hosted := newCallers()
	client := newVaultClient(t, hosted, "token-a")

	res, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "samples/A.md"}))

	if err != nil || res.Msg.GetError() != v1.ErrorCode_ERROR_CODE_NO_VAULT {
		t.Errorf("the answer was %v (%v)", res.Msg, err)
	}
}

func TestGetNoteRefs_ReturnsIndexedNotesAndLinks(t *testing.T) {
	client, _ := newVaultFixture(t)
	ctx := t.Context()

	refs, err := client.ListNoteRefs(ctx, connect.NewRequest(&v1.ListNoteRefsRequest{Paths: []string{"samples/A.md", "gone.md"}}))
	if err != nil || len(refs.Msg.GetNotes()) != 1 || refs.Msg.GetNotes()[0].GetTitle() != "Title of samples/A.md" {
		t.Errorf("the notes were %v (%v)", refs.Msg, err)
	}
	types, err := client.ListNoteTypes(ctx, connect.NewRequest(&v1.ListNoteTypesRequest{Paths: []string{"samples/A.md"}}))
	if err != nil || types.Msg.GetTypes()["samples/A.md"] != "sample" {
		t.Errorf("the types were %v (%v)", types.Msg, err)
	}
	paths, err := client.ListPathsOfType(ctx, connect.NewRequest(&v1.ListPathsOfTypeRequest{Type: "sample"}))
	if err != nil || !slices.Equal(paths.Msg.GetPaths(), []string{"Animal.md", "samples/A.md"}) {
		t.Errorf("the paths were %v (%v)", paths.Msg, err)
	}
	links, err := client.ListLinks(ctx, connect.NewRequest(&v1.ListLinksRequest{Path: "samples/A.md"}))
	if err != nil || len(links.Msg.GetLinks()) != 1 {
		t.Fatalf("the links were %v (%v)", links.Msg, err)
	}
	link := links.Msg.GetLinks()[0]
	if link.GetFrom() != "samples/A.md" || link.GetTo() != "Animal.md" || link.GetToVault() != "v1" || link.GetLinkType() != "preset" ||
		link.GetTargetScheme() != "name" || link.GetTargetValue() != "Animal" || link.GetLabel() != "by" || !link.GetIsAmbiguous() {
		t.Errorf("the link came out as %v", link)
	}
	back, err := client.ListBacklinks(ctx, connect.NewRequest(&v1.HostLinkServiceListBacklinksRequest{Path: "Animal.md"}))
	if err != nil || len(back.Msg.GetLinks()) != 1 {
		t.Errorf("the backlinks were %v (%v)", back.Msg, err)
	}
	resolved, err := client.ResolveLinks(ctx, connect.NewRequest(&v1.ResolveLinksRequest{Path: "samples/A.md", Addresses: []string{"Animal"}}))
	if err != nil || resolved.Msg.GetLinks()["Animal"].GetTo() != "Animal.md" {
		t.Errorf("the resolved links were %v (%v)", resolved.Msg, err)
	}
	problems, err := client.ListParseProblems(ctx, connect.NewRequest(&v1.ListParseProblemsRequest{}))
	if err != nil || len(problems.Msg.GetProblems()) != 1 || problems.Msg.GetProblems()[0].GetDetail() != "an entry has no role" {
		t.Errorf("the problems were %v (%v)", problems.Msg, err)
	}
}

func TestACallIsRefusedWithoutThePermissionItNeeds(t *testing.T) {
	vault := newMemoryVault(map[string]string{"a.md": "a"})
	hosted := newCallers()
	hosted.vault = vaultOf(vault)
	client := newVaultClient(t, hosted, "token-b")

	_, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "a.md"}))
	_, writeErr := client.CreateFile(t.Context(), connect.NewRequest(&v1.CreateFileRequest{Path: "b.md"}))
	_, anonymous := newVaultClient(t, hosted, "").GetHostVault(t.Context(), connect.NewRequest(&v1.GetHostVaultRequest{}))

	if got := readCode(t, err); got != connect.CodePermissionDenied {
		t.Errorf("a read without vault:read is %v", got)
	}
	if got := readCode(t, writeErr); got != connect.CodePermissionDenied {
		t.Errorf("a write without vault:write is %v", got)
	}
	if got := readCode(t, anonymous); got != connect.CodeUnauthenticated {
		t.Errorf("a call with no token is %v", got)
	}
	if !slices.Contains(hosted.checked, "md.numen.b "+manifest.PermissionVaultRead) || !slices.Contains(hosted.checked, "md.numen.b "+manifest.PermissionVaultWrite) {
		t.Errorf("the permissions asked were %v", hosted.checked)
	}
}

func TestTheNotesFiledBelowANameAreListed(t *testing.T) {
	client, _ := newVaultFixture(t)

	res, err := client.ListNamedPaths(t.Context(), connect.NewRequest(&v1.ListNamedPathsRequest{Name: "Animal"}))

	if err != nil || !slices.Equal(res.Msg.GetPaths(), []string{"Animal.md"}) {
		t.Errorf("the paths were %v (%v)", res.Msg.GetPaths(), err)
	}
}

func TestABacklinkCarriesItsRoleAndTheRoleFromTheNoteAskedAbout(t *testing.T) {
	client, _ := newVaultFixture(t)

	res, err := client.ListBacklinks(t.Context(), connect.NewRequest(&v1.HostLinkServiceListBacklinksRequest{Path: "Animal.md"}))

	if err != nil || len(res.Msg.GetLinks()) != 1 {
		t.Fatalf("the answer was %v (%v)", res.Msg, err)
	}
	link := res.Msg.GetLinks()[0]
	if link.GetRole() != v1.Role_ROLE_PARENT || link.GetSourceRole() != v1.Role_ROLE_CHILD ||
		link.GetMarker() != v1.Marker_MARKER_ARROW || link.GetNote() != "because" {
		t.Errorf("the backlink came out as %v", link)
	}
}

func (m *memoryVault) EnsureIndexed(context.Context) (plugin.IndexState, error) {
	return plugin.IndexState{Status: plugin.Indexed}, nil
}

func (m *memoryVault) Lock(context.Context) (func(), error) { return func() {}, nil }

func (m *memoryVault) ScanFiles(context.Context, []string) error { return nil }

func TestVaultReadAndVaultWriteAreGrantedApart(t *testing.T) {
	cases := []struct {
		granted         string
		isRead, isWrite bool
	}{
		{granted: manifest.PermissionVaultRead, isRead: true},
		{granted: manifest.PermissionVaultWrite, isWrite: true},
	}
	for _, c := range cases {
		hosted := newCallers()
		hosted.vault = vaultOf(newMemoryVault(map[string]string{"a.md": "a"}))
		hosted.granted, hosted.confined = []string{c.granted}, true
		client := newVaultClient(t, hosted, "token-a")

		_, readErr := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "a.md"}))
		_, writeErr := client.CreateFile(t.Context(), connect.NewRequest(&v1.CreateFileRequest{Path: "b.md"}))

		if c.isRead != (readErr == nil) || (!c.isRead && connect.CodeOf(readErr) != connect.CodePermissionDenied) {
			t.Errorf("with only %s a read was answered %v", c.granted, readErr)
		}
		if c.isWrite != (writeErr == nil) || (!c.isWrite && connect.CodeOf(writeErr) != connect.CodePermissionDenied) {
			t.Errorf("with only %s a write was answered %v", c.granted, writeErr)
		}
	}
}

// memoryCache keeps the cache entries of the plugins by plugin and name.
type memoryCache struct {
	mu      sync.Mutex
	entries map[string][]byte
}

func (c *memoryCache) ReadCache(_ context.Context, pluginID, name string) ([]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	content, found := c.entries[pluginID+"/"+name]
	return content, found, nil
}

func (c *memoryCache) WriteCache(_ context.Context, pluginID, name string, content []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[pluginID+"/"+name] = content
	return nil
}

func TestThePluginsCacheNeedsNoVaultPermission(t *testing.T) {
	hosted := newCallers()
	hosted.vault = vaultOf(&memoryCache{entries: map[string][]byte{}})
	hosted.confined = true
	client := newVaultClient(t, hosted, "token-a")
	ctx := t.Context()

	_, writeErr := client.WriteCache(ctx, connect.NewRequest(&v1.WriteCacheRequest{Name: "schedules", Content: []byte("{}")}))
	read, readErr := client.ReadCache(ctx, connect.NewRequest(&v1.ReadCacheRequest{Name: "schedules"}))
	_, anonymous := newVaultClient(t, hosted, "").ReadCache(ctx, connect.NewRequest(&v1.ReadCacheRequest{Name: "schedules"}))

	if writeErr != nil || readErr != nil || string(read.Msg.GetContent()) != "{}" {
		t.Errorf("a plugin without vault permissions wrote its cache (%v) and read %v (%v)", writeErr, read, readErr)
	}
	if got := readCode(t, anonymous); got != connect.CodeUnauthenticated {
		t.Errorf("a cache call with no token is %v", got)
	}
}

func (m *memoryVault) WriteFrontmatter(context.Context, []plugin.FrontmatterWrite) ([]plugin.FrontmatterWritten, error) {
	return nil, errors.New("not implemented")
}
