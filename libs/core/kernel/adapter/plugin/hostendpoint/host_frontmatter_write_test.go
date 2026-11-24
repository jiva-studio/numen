package hostendpoint

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// writingVault is the vault of rivers that writes frontmatter as the host does: a baseline the note
// moved past is the core's stale error, a note that is not there fs.ErrNotExist. It records every call.
type writingVault struct {
	*linkingVault
	writes [][]plugin.FrontmatterWrite
}

func newWritingVault() *writingVault {
	return &writingVault{linkingVault: newLinkingVault()}
}

func (w *writingVault) WriteFrontmatter(_ context.Context, writes []plugin.FrontmatterWrite) ([]plugin.FrontmatterWritten, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writes = append(w.writes, writes)
	out := make([]plugin.FrontmatterWritten, len(writes))
	for i, one := range writes {
		if err := w.checkBaseline(one); err != nil {
			out[i] = plugin.FrontmatterWritten{Err: err}
			continue
		}
		stat, err := w.write(one.Path)
		out[i] = plugin.FrontmatterWritten{Stat: stat, Err: err}
	}
	return out, nil
}

func (w *writingVault) checkBaseline(one plugin.FrontmatterWrite) error {
	if one.Baseline.IsZero() {
		_, err := w.stat(one.Path)
		if err != nil {
			return fmt.Errorf("read %s: %w", one.Path, err)
		}
		return nil
	}
	return w.checkNote(one.Path, one.Baseline)
}

func newWritingClient(t *testing.T, vault *writingVault, granted ...string) HostVaultClient {
	t.Helper()
	hostCallers := newCallers()
	if vault != nil {
		hostCallers.vault = vaultOf(vault)
	}
	endpoint := &Endpoint{callers: hostCallers.over(nil, granted...)}
	mux := http.NewServeMux()
	vaultapi.Mount(mux, endpoint, endpoint.callers.Lent)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return NewHostVaultClient(server.Client(), server.URL, connect.WithInterceptors(transport.CreateAuthInterceptor("token-a")))
}

func TestWriteFrontmatterIsDeniedWithoutVaultWrite(t *testing.T) {
	vault := newWritingVault()
	client := newWritingClient(t, vault, manifest.PermissionVaultRead)

	_, err := client.WriteFrontmatter(t.Context(), connect.NewRequest(&v1.WriteFrontmatterRequest{Notes: []*v1.NoteFrontmatterWrite{
		{Path: "Danube.md", Values: []byte(`{"tags":["river"]}`), Baseline: vault.readBaseline(t, "Danube.md")},
	}}))

	if !isDenied(err) {
		t.Errorf("WriteFrontmatter with only vault:read returned %v, want permission denied", err)
	}
	if len(vault.writes) != 0 {
		t.Errorf("a denied call was sent to the vault: %v", vault.writes)
	}
}

func TestWriteFrontmatterWithoutAVaultReturnsNoVault(t *testing.T) {
	client := newWritingClient(t, nil, manifest.PermissionVaultWrite)

	res, err := client.WriteFrontmatter(t.Context(), connect.NewRequest(&v1.WriteFrontmatterRequest{Notes: []*v1.NoteFrontmatterWrite{
		{Path: "Danube.md", Values: []byte(`{"tags":["river"]}`)},
	}}))

	if err != nil || res.Msg.GetError() != v1.ErrorCode_ERROR_CODE_NO_VAULT || len(res.Msg.GetResults()) != 0 {
		t.Errorf("WriteFrontmatter returned %v (%v), want NO_VAULT and no results", res, err)
	}
}

func TestWriteFrontmatterPassesThePathsTheDecodedValuesAndTheBaselinesInOneCall(t *testing.T) {
	vault := newWritingVault()
	client := newWritingClient(t, vault, manifest.PermissionVaultWrite)
	danube := vault.readBaseline(t, "Danube.md")

	res, err := client.WriteFrontmatter(t.Context(), connect.NewRequest(&v1.WriteFrontmatterRequest{Notes: []*v1.NoteFrontmatterWrite{
		{Path: "Danube.md", Values: []byte(`{"tags":["river","lake"],"status":null,"rating":3}`), Baseline: danube},
		{Path: "Deltas.md", Values: []byte(`{}`)},
	}}))
	if err != nil {
		t.Fatal(err)
	}

	if len(vault.writes) != 1 || len(vault.writes[0]) != 2 {
		t.Fatalf("the vault received %v, want the two notes in one call", vault.writes)
	}
	first := vault.writes[0][0]
	tags, isList := first.Values["tags"].([]any)
	status, hasStatus := first.Values["status"]
	if first.Path != "Danube.md" || !isList || len(tags) != 2 || tags[0] != "river" || tags[1] != "lake" || !hasStatus || status != nil || first.Values["rating"] != float64(3) {
		t.Errorf("the first note was sent as %+v, want the tags, the rating and the status as nil", first)
	}
	if first.Baseline.Size != danube.GetSize() || first.Baseline.ModTime.UnixNano() != danube.GetMtime() {
		t.Errorf("the baseline was sent as %+v, want %v", first.Baseline, danube)
	}
	if !vault.writes[0][1].Baseline.IsZero() {
		t.Errorf("a note with no baseline was sent the baseline %+v", vault.writes[0][1].Baseline)
	}
	results := res.Msg.GetResults()
	if res.Msg.GetError() != 0 || len(results) != 2 {
		t.Fatalf("WriteFrontmatter returned error %v with %d results, want 2 results", res.Msg.GetError(), len(results))
	}
	for i, path := range []string{"Danube.md", "Deltas.md"} {
		if r := results[i]; r.GetError() != 0 || r.GetDetail() != "" || r.GetStat().GetPath() != path || r.GetStat().GetSize() == 0 {
			t.Errorf("result %d is %v, want the file stat of %s after the write and no error", i, r, path)
		}
	}
}

func TestWriteFrontmatterOverAStaleBaselineReturnsConflictAndWritesTheOthers(t *testing.T) {
	vault := newWritingVault()
	client := newWritingClient(t, vault, manifest.PermissionVaultWrite)
	stale := vault.readBaseline(t, "Danube.md")
	stale.Size++

	res, err := client.WriteFrontmatter(t.Context(), connect.NewRequest(&v1.WriteFrontmatterRequest{Notes: []*v1.NoteFrontmatterWrite{
		{Path: "Danube.md", Values: []byte(`{"tags":"river"}`), Baseline: stale},
		{Path: "Deltas.md", Values: []byte(`{"tags":"delta"}`), Baseline: vault.readBaseline(t, "Deltas.md")},
		{Path: "Volga.md", Values: []byte(`{"tags":"river"}`)},
	}}))
	if err != nil {
		t.Fatal(err)
	}

	results := res.Msg.GetResults()
	if len(results) != 3 {
		t.Fatalf("WriteFrontmatter returned %v, want 3 results", res.Msg)
	}
	if results[0].GetError() != v1.ErrorCode_ERROR_CODE_CONFLICT || !strings.Contains(results[0].GetDetail(), "Danube.md") {
		t.Errorf("the stale note returned %v, want CONFLICT naming Danube.md", results[0])
	}
	if results[1].GetError() != 0 || results[1].GetStat().GetPath() != "Deltas.md" {
		t.Errorf("the note after the stale one returned %v, want it written", results[1])
	}
	if results[2].GetError() != v1.ErrorCode_ERROR_CODE_NOT_FOUND || !strings.Contains(results[2].GetDetail(), "Volga.md") {
		t.Errorf("the missing note returned %v, want NOT_FOUND naming Volga.md", results[2])
	}
}

func TestWriteFrontmatterRejectsMoreThanAThousandNotes(t *testing.T) {
	vault := newWritingVault()
	client := newWritingClient(t, vault, manifest.PermissionVaultWrite)
	notes := make([]*v1.NoteFrontmatterWrite, 1001)
	for i := range notes {
		notes[i] = &v1.NoteFrontmatterWrite{Path: "Danube.md", Values: []byte(`{}`)}
	}

	_, err := client.WriteFrontmatter(t.Context(), connect.NewRequest(&v1.WriteFrontmatterRequest{Notes: notes}))

	if readCode(t, err) != connect.CodeInvalidArgument {
		t.Errorf("1001 notes returned %v, want invalid argument", err)
	}
	if len(vault.writes) != 0 {
		t.Errorf("a rejected call was sent to the vault with %d calls", len(vault.writes))
	}
}

func TestWriteFrontmatterAcceptsAThousandNotes(t *testing.T) {
	vault := newWritingVault()
	client := newWritingClient(t, vault, manifest.PermissionVaultWrite)
	notes := make([]*v1.NoteFrontmatterWrite, 1000)
	for i := range notes {
		notes[i] = &v1.NoteFrontmatterWrite{Path: "Danube.md", Values: []byte(`{}`)}
	}

	res, err := client.WriteFrontmatter(t.Context(), connect.NewRequest(&v1.WriteFrontmatterRequest{Notes: notes}))

	if err != nil || len(res.Msg.GetResults()) != 1000 {
		t.Fatalf("1000 notes returned %v (%v), want 1000 results", res, err)
	}
}

func TestWriteFrontmatterRejectsValuesThatAreNotAJSONObjectAndWritesNothing(t *testing.T) {
	vault := newWritingVault()
	client := newWritingClient(t, vault, manifest.PermissionVaultWrite)

	_, err := client.WriteFrontmatter(t.Context(), connect.NewRequest(&v1.WriteFrontmatterRequest{Notes: []*v1.NoteFrontmatterWrite{
		{Path: "Danube.md", Values: []byte(`["tags"]`)},
	}}))

	if readCode(t, err) != connect.CodeInvalidArgument {
		t.Errorf("values that are an array returned %v, want invalid argument", err)
	}
	if len(vault.writes) != 0 {
		t.Errorf("a rejected call was sent to the vault: %v", vault.writes)
	}
}
