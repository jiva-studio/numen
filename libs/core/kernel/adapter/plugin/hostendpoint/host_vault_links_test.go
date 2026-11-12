package hostendpoint

import (
	"context"
	"fmt"
	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// linkUpdate is one UpdateLinks call the linking vault received.
type linkUpdate struct {
	from     string
	to       domain.Address
	change   domain.Link
	baseline domain.FileStat
}

// linkRemoval is one RemoveLinks call the linking vault received.
type linkRemoval struct {
	from     string
	to       domain.Address
	role     domain.LinkRole
	baseline domain.FileStat
}

// linkingVault is the vault of rivers that edits frontmatter links as the host does: a baseline
// the note moved past is vaultport.ErrStale, a note that is not there fs.ErrNotExist, a target the
// note has no link to the core's "no link to <to> is written here". It records every call.
type linkingVault struct {
	*fakeNoteVault
	additions [][]plugin.LinkAddition
	updates   []linkUpdate
	removals  []linkRemoval
	links     map[string][]domain.Address
}

func newLinkingVault() *linkingVault {
	return &linkingVault{
		fakeNoteVault: newFakeNoteVault(),
		links:         map[string][]domain.Address{"Deltas.md": {{Scheme: domain.SchemeName, Value: "Danube"}}},
	}
}

func (l *linkingVault) checkNote(from string, baseline domain.FileStat) error {
	now, err := l.stat(from)
	if err != nil {
		return fmt.Errorf("read %s: %w", from, err)
	}
	if now.Size != baseline.Size || !now.ModTime.Equal(baseline.ModTime) {
		return fmt.Errorf("write %s: %w", from, edit.ErrStale)
	}
	return nil
}

func (l *linkingVault) write(from string) (domain.FileStat, error) {
	l.files[from] += "\n"
	l.stamp++
	return l.stat(from)
}

func (l *linkingVault) hasLink(from string, to domain.Address) bool {
	return slices.Contains(l.links[from], to)
}

func (l *linkingVault) AddLinks(_ context.Context, additions []plugin.LinkAddition) ([]plugin.LinkAdded, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.additions = append(l.additions, additions)
	out := make([]plugin.LinkAdded, len(additions))
	rejected := map[string]error{}
	for _, one := range additions {
		if _, isChecked := rejected[one.From]; !isChecked {
			rejected[one.From] = l.checkNote(one.From, one.Baseline)
		}
	}
	savedStats := map[string]domain.FileStat{}
	for i, one := range additions {
		if err := rejected[one.From]; err != nil {
			out[i] = plugin.LinkAdded{Err: err}
			continue
		}
		if _, isSaved := savedStats[one.From]; !isSaved {
			stat, err := l.write(one.From)
			if err != nil {
				out[i] = plugin.LinkAdded{Err: err}
				continue
			}
			savedStats[one.From] = stat
		}
		l.links[one.From] = append(l.links[one.From], one.Link.Target)
		out[i] = plugin.LinkAdded{Stat: savedStats[one.From]}
	}
	return out, nil
}

func (l *linkingVault) UpdateLinks(_ context.Context, from string, to domain.Address, change domain.Link, baseline domain.FileStat) (domain.FileStat, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.updates = append(l.updates, linkUpdate{from: from, to: to, change: change, baseline: baseline})
	if err := l.checkNote(from, baseline); err != nil {
		return domain.FileStat{}, err
	}
	if !l.hasLink(from, to) {
		return domain.FileStat{}, fmt.Errorf("no link to %s is written here", to.GetWritten())
	}
	return l.write(from)
}

func (l *linkingVault) RemoveLinks(_ context.Context, from string, to domain.Address, role domain.LinkRole, baseline domain.FileStat) (domain.FileStat, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.removals = append(l.removals, linkRemoval{from: from, to: to, role: role, baseline: baseline})
	if err := l.checkNote(from, baseline); err != nil {
		return domain.FileStat{}, err
	}
	if !l.hasLink(from, to) {
		return domain.FileStat{}, fmt.Errorf("no link to %s is written here", to.GetWritten())
	}
	return l.write(from)
}

func (l *linkingVault) readBaseline(t *testing.T, from string) *v1.HostFileStat {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	now, err := l.stat(from)
	if err != nil {
		t.Fatal(err)
	}
	return writeFileStat(now)
}

func newLinksClient(t *testing.T, vault *linkingVault, granted ...string) numenv1connect.HostVaultServiceClient {
	t.Helper()
	hostCallers := newCallers(nil)
	if vault != nil {
		hostCallers.vault = vaultOf(vault)
	}
	endpoint := &Endpoint{callers: hostCallers.over(nil, granted...)}
	mux := http.NewServeMux()
	mux.Handle(vaultapi.NewHandler(endpoint, endpoint.callers.Lent))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewHostVaultServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor("token-a")))
}

func TestLinkEditsAreDeniedWithoutVaultWrite(t *testing.T) {
	vault := newLinkingVault()
	client := newLinksClient(t, vault, manifest.PermissionVaultRead)
	ctx := t.Context()

	_, addErr := client.AddLinks(ctx, connect.NewRequest(&v1.HostVaultServiceAddLinksRequest{Links: []*v1.HostLinkAddition{
		{From: "Danube.md", Baseline: vault.readBaseline(t, "Danube.md"), To: "Deltas", Role: v1.Role_ROLE_CHILD},
	}}))
	_, updateErr := client.UpdateLinks(ctx, connect.NewRequest(&v1.HostVaultServiceUpdateLinksRequest{
		From: "Deltas.md", To: "Danube", Baseline: vault.readBaseline(t, "Deltas.md"), Label: "flows into",
	}))
	_, removeErr := client.RemoveLinks(ctx, connect.NewRequest(&v1.HostVaultServiceRemoveLinksRequest{
		From: "Deltas.md", To: "Danube", Baseline: vault.readBaseline(t, "Deltas.md"),
	}))

	for name, err := range map[string]error{"AddLinks": addErr, "UpdateLinks": updateErr, "RemoveLinks": removeErr} {
		if !isDenied(err) {
			t.Errorf("%s with only vault:read returned %v, want permission denied", name, err)
		}
	}
	if len(vault.additions) != 0 || len(vault.updates) != 0 || len(vault.removals) != 0 {
		t.Errorf("a denied call was sent to the vault: %v %v %v", vault.additions, vault.updates, vault.removals)
	}
}

func TestLinkEditsWithoutAVaultReturnNoVault(t *testing.T) {
	client := newLinksClient(t, nil, manifest.PermissionVaultWrite)
	ctx := t.Context()

	added, addErr := client.AddLinks(ctx, connect.NewRequest(&v1.HostVaultServiceAddLinksRequest{Links: []*v1.HostLinkAddition{
		{From: "Danube.md", To: "Deltas", Role: v1.Role_ROLE_CHILD},
	}}))
	updated, updateErr := client.UpdateLinks(ctx, connect.NewRequest(&v1.HostVaultServiceUpdateLinksRequest{From: "Deltas.md", To: "Danube"}))
	removed, removeErr := client.RemoveLinks(ctx, connect.NewRequest(&v1.HostVaultServiceRemoveLinksRequest{From: "Deltas.md", To: "Danube"}))

	if addErr != nil || added.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_VAULT || len(added.Msg.GetResults()) != 0 {
		t.Errorf("AddLinks returned %v (%v), want NO_VAULT and no results", added, addErr)
	}
	if updateErr != nil || updated.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_VAULT {
		t.Errorf("UpdateLinks returned %v (%v), want NO_VAULT", updated, updateErr)
	}
	if removeErr != nil || removed.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_VAULT {
		t.Errorf("RemoveLinks returned %v (%v), want NO_VAULT", removed, removeErr)
	}
}

func TestAddLinksSendsOneCallAndReturnsOneResultPerLink(t *testing.T) {
	vault := newLinkingVault()
	client := newLinksClient(t, vault, manifest.PermissionVaultWrite)
	danube := vault.readBaseline(t, "Danube.md")
	rhine := vault.readBaseline(t, "Rivers/Rhine.md")

	res, err := client.AddLinks(t.Context(), connect.NewRequest(&v1.HostVaultServiceAddLinksRequest{Links: []*v1.HostLinkAddition{
		{From: "Danube.md", Baseline: danube, To: "Deltas", Role: v1.Role_ROLE_CHILD, Type: "delta", Label: "ends in", Note: "the river ends in the delta"},
		{From: "Rivers/Rhine.md", Baseline: rhine, To: "note://01JRIVERDanube", Role: v1.Role_ROLE_JUMP},
		{From: "Danube.md", Baseline: danube, To: "Rivers/Rhine", Role: v1.Role_ROLE_REF},
	}}))
	if err != nil {
		t.Fatal(err)
	}

	if len(vault.additions) != 1 || len(vault.additions[0]) != 3 {
		t.Fatalf("the vault received the additions %v, want the three in one call", vault.additions)
	}
	got := vault.additions[0]
	want := []plugin.LinkAddition{
		{From: "Danube.md", Baseline: readFileStat(danube), Link: domain.Link{
			Target: domain.Address{Scheme: domain.SchemeName, Value: "Deltas"}, Role: domain.RoleChild,
			Type: "delta", Label: "ends in", Why: "the river ends in the delta",
		}},
		{From: "Rivers/Rhine.md", Baseline: readFileStat(rhine), Link: domain.Link{
			Target: domain.Address{Scheme: domain.SchemeNote, Value: "01JRIVERDanube"}, Role: domain.RoleJump,
		}},
		{From: "Danube.md", Baseline: readFileStat(danube), Link: domain.Link{
			Target: domain.Address{Scheme: domain.SchemeName, Value: "Rivers/Rhine"}, Role: domain.RoleRef,
		}},
	}
	for i := range want {
		if got[i].From != want[i].From || got[i].Link != want[i].Link ||
			got[i].Baseline.Size != want[i].Baseline.Size || !got[i].Baseline.ModTime.Equal(want[i].Baseline.ModTime) {
			t.Errorf("addition %d was sent to the vault as %+v, want %+v", i, got[i], want[i])
		}
	}
	results := res.Msg.GetResults()
	if res.Msg.GetError() != 0 || len(results) != 3 {
		t.Fatalf("AddLinks returned error %v with %d results, want 3 results", res.Msg.GetError(), len(results))
	}
	for i, from := range []string{"Danube.md", "Rivers/Rhine.md", "Danube.md"} {
		if r := results[i]; r.GetError() != 0 || r.GetDetail() != "" || r.GetStat().GetPath() != from || r.GetStat().GetSize() == 0 {
			t.Errorf("result %d is %v, want the file stat of %s after the write and no error", i, r, from)
		}
	}
	if results[0].GetStat().GetModTime() != results[2].GetStat().GetModTime() {
		t.Errorf("the two links of Danube.md returned the file stats %v and %v, want the one write", results[0].GetStat(), results[2].GetStat())
	}
}

func TestAddLinksOverStaleBaselineReturnsConflictAndWritesOthers(t *testing.T) {
	vault := newLinkingVault()
	client := newLinksClient(t, vault, manifest.PermissionVaultWrite)
	outdated := vault.readBaseline(t, "Danube.md")
	outdated.Size++
	rhine := vault.readBaseline(t, "Rivers/Rhine.md")

	res, err := client.AddLinks(t.Context(), connect.NewRequest(&v1.HostVaultServiceAddLinksRequest{Links: []*v1.HostLinkAddition{
		{From: "Danube.md", Baseline: outdated, To: "Deltas", Role: v1.Role_ROLE_CHILD},
		{From: "Rivers/Rhine.md", Baseline: rhine, To: "Danube", Role: v1.Role_ROLE_JUMP},
		{From: "Danube.md", Baseline: outdated, To: "Rivers/Rhine", Role: v1.Role_ROLE_REF},
	}}))
	if err != nil {
		t.Fatal(err)
	}

	results := res.Msg.GetResults()
	if res.Msg.GetError() != 0 || len(results) != 3 {
		t.Fatalf("AddLinks returned error %v with %d results, want 3 results", res.Msg.GetError(), len(results))
	}
	for _, i := range []int{0, 2} {
		if r := results[i]; r.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_CONFLICT || r.GetStat() != nil ||
			!strings.Contains(r.GetDetail(), "Danube.md") {
			t.Errorf("result %d is %v, want CONFLICT with the core's text and no file stat", i, r)
		}
	}
	if r := results[1]; r.GetError() != 0 || r.GetStat().GetPath() != "Rivers/Rhine.md" {
		t.Errorf("the link of Rivers/Rhine.md returned %v, want it written", r)
	}
}

func TestAddLinksToMissingNoteReturnsNotFoundWithTheDetail(t *testing.T) {
	vault := newLinkingVault()
	client := newLinksClient(t, vault, manifest.PermissionVaultWrite)

	res, err := client.AddLinks(t.Context(), connect.NewRequest(&v1.HostVaultServiceAddLinksRequest{Links: []*v1.HostLinkAddition{
		{From: "Volga.md", Baseline: &v1.HostFileStat{Path: "Volga.md", Size: 8, ModTime: time.Unix(0, 100).UnixNano()}, To: "Danube", Role: v1.Role_ROLE_JUMP},
	}}))
	if err != nil {
		t.Fatal(err)
	}

	if r := res.Msg.GetResults(); len(r) != 1 || r[0].GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND ||
		!strings.Contains(r[0].GetDetail(), "Volga.md") {
		t.Errorf("AddLinks returned %v, want one result NOT_FOUND naming Volga.md", res.Msg)
	}
}

func TestAddLinksRejectsMoreThanFiftyLinks(t *testing.T) {
	vault := newLinkingVault()
	client := newLinksClient(t, vault, manifest.PermissionVaultWrite)
	baseline := vault.readBaseline(t, "Danube.md")
	links := make([]*v1.HostLinkAddition, 51)
	for i := range links {
		links[i] = &v1.HostLinkAddition{From: "Danube.md", Baseline: baseline, To: fmt.Sprintf("Tributary %d", i), Role: v1.Role_ROLE_CHILD}
	}

	_, err := client.AddLinks(t.Context(), connect.NewRequest(&v1.HostVaultServiceAddLinksRequest{Links: links}))

	if readCode(t, err) != connect.CodeInvalidArgument {
		t.Errorf("51 links returned %v, want invalid argument", err)
	}
	if len(vault.additions) != 0 {
		t.Errorf("a rejected call was sent to the vault with %d calls", len(vault.additions))
	}
}

func TestAddLinksAcceptsFiftyLinks(t *testing.T) {
	vault := newLinkingVault()
	client := newLinksClient(t, vault, manifest.PermissionVaultWrite)
	baseline := vault.readBaseline(t, "Danube.md")
	links := make([]*v1.HostLinkAddition, 50)
	for i := range links {
		links[i] = &v1.HostLinkAddition{From: "Danube.md", Baseline: baseline, To: fmt.Sprintf("Tributary %d", i), Role: v1.Role_ROLE_CHILD}
	}

	res, err := client.AddLinks(t.Context(), connect.NewRequest(&v1.HostVaultServiceAddLinksRequest{Links: links}))

	if err != nil || len(res.Msg.GetResults()) != 50 {
		t.Fatalf("50 links returned %v (%v), want 50 results", res, err)
	}
}

func TestUpdateLinksPassesTheChangeAndReturnsTheFileStat(t *testing.T) {
	vault := newLinkingVault()
	client := newLinksClient(t, vault, manifest.PermissionVaultWrite)
	baseline := vault.readBaseline(t, "Deltas.md")

	res, err := client.UpdateLinks(t.Context(), connect.NewRequest(&v1.HostVaultServiceUpdateLinksRequest{
		From: "Deltas.md", To: "Danube", Baseline: baseline,
		Role: v1.Role_ROLE_JUMP, Type: "outflow", Label: "flows from", Note: "the delta is the mouth of the river",
	}))
	if err != nil {
		t.Fatal(err)
	}

	if len(vault.updates) != 1 {
		t.Fatalf("the vault received the updates %v, want one", vault.updates)
	}
	got := vault.updates[0]
	wantChange := domain.Link{Role: domain.RoleJump, Type: "outflow", Label: "flows from", Why: "the delta is the mouth of the river"}
	if got.from != "Deltas.md" || got.to != (domain.Address{Scheme: domain.SchemeName, Value: "Danube"}) ||
		got.change.Role != wantChange.Role || got.change.Type != wantChange.Type || got.change.Label != wantChange.Label || got.change.Why != wantChange.Why ||
		got.baseline.Size != baseline.GetSize() || got.baseline.ModTime.UnixNano() != baseline.GetModTime() {
		t.Errorf("the update was sent to the vault as %+v, want Deltas.md to name://Danube with %+v over the baseline %v", got, wantChange, baseline)
	}
	if res.Msg.GetError() != 0 || res.Msg.GetDetail() != "" || res.Msg.GetStat().GetPath() != "Deltas.md" ||
		res.Msg.GetStat().GetModTime() == baseline.GetModTime() {
		t.Errorf("UpdateLinks returned %v, want the file stat of Deltas.md after the write", res.Msg)
	}
}

func TestUpdateLinksOverAStaleBaselineReturnsConflict(t *testing.T) {
	vault := newLinkingVault()
	client := newLinksClient(t, vault, manifest.PermissionVaultWrite)
	outdated := vault.readBaseline(t, "Deltas.md")
	outdated.ModTime++

	res, err := client.UpdateLinks(t.Context(), connect.NewRequest(&v1.HostVaultServiceUpdateLinksRequest{
		From: "Deltas.md", To: "Danube", Baseline: outdated, Label: "flows from",
	}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_CONFLICT || res.Msg.GetStat() != nil || res.Msg.GetDetail() == "" {
		t.Errorf("UpdateLinks over a stale baseline returned %v, want CONFLICT with the core's text", res.Msg)
	}
}

func TestUpdateLinksOfUnlinkedTargetReturnsTheCoreText(t *testing.T) {
	vault := newLinkingVault()
	client := newLinksClient(t, vault, manifest.PermissionVaultWrite)

	res, err := client.UpdateLinks(t.Context(), connect.NewRequest(&v1.HostVaultServiceUpdateLinksRequest{
		From: "Deltas.md", To: "Volga", Baseline: vault.readBaseline(t, "Deltas.md"), Label: "flows from",
	}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INTERNAL || res.Msg.GetDetail() != "no link to Volga is written here" {
		t.Errorf("UpdateLinks of a missing link returned %v, want INTERNAL with the detail naming name://Volga", res.Msg)
	}
}

func TestRemoveLinksPassesTheRoleAndReturnsTheFileStat(t *testing.T) {
	vault := newLinkingVault()
	client := newLinksClient(t, vault, manifest.PermissionVaultWrite)
	baseline := vault.readBaseline(t, "Deltas.md")

	res, err := client.RemoveLinks(t.Context(), connect.NewRequest(&v1.HostVaultServiceRemoveLinksRequest{
		From: "Deltas.md", To: "Danube", Baseline: baseline, Role: v1.Role_ROLE_PARENT,
	}))
	if err != nil {
		t.Fatal(err)
	}

	if len(vault.removals) != 1 {
		t.Fatalf("the vault received the removals %v, want one", vault.removals)
	}
	got := vault.removals[0]
	if got.from != "Deltas.md" || got.to != (domain.Address{Scheme: domain.SchemeName, Value: "Danube"}) || got.role != domain.RoleParent ||
		got.baseline.Size != baseline.GetSize() || got.baseline.ModTime.UnixNano() != baseline.GetModTime() {
		t.Errorf("the removal was sent to the vault as %+v, want Deltas.md to name://Danube of role parent", got)
	}
	if res.Msg.GetError() != 0 || res.Msg.GetDetail() != "" || res.Msg.GetStat().GetPath() != "Deltas.md" {
		t.Errorf("RemoveLinks returned %v, want the file stat of Deltas.md after the write", res.Msg)
	}
}

func TestRemoveLinksWithoutARolePassesNoRole(t *testing.T) {
	vault := newLinkingVault()
	client := newLinksClient(t, vault, manifest.PermissionVaultWrite)

	_, err := client.RemoveLinks(t.Context(), connect.NewRequest(&v1.HostVaultServiceRemoveLinksRequest{
		From: "Deltas.md", To: "Danube", Baseline: vault.readBaseline(t, "Deltas.md"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	if len(vault.removals) != 1 || vault.removals[0].role != "" {
		t.Errorf("the removal was sent to the vault as %+v, want no role so every role is removed", vault.removals)
	}
}

func TestRemoveLinksOverAStaleBaselineReturnsConflict(t *testing.T) {
	vault := newLinkingVault()
	client := newLinksClient(t, vault, manifest.PermissionVaultWrite)
	outdated := vault.readBaseline(t, "Deltas.md")
	outdated.Size--

	res, err := client.RemoveLinks(t.Context(), connect.NewRequest(&v1.HostVaultServiceRemoveLinksRequest{
		From: "Deltas.md", To: "Danube", Baseline: outdated,
	}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_CONFLICT || res.Msg.GetStat() != nil || res.Msg.GetDetail() == "" {
		t.Errorf("RemoveLinks over a stale baseline returned %v, want CONFLICT with the core's text", res.Msg)
	}
}

func TestRemoveLinksOfUnlinkedTargetReturnsTheCoreText(t *testing.T) {
	vault := newLinkingVault()
	client := newLinksClient(t, vault, manifest.PermissionVaultWrite)

	res, err := client.RemoveLinks(t.Context(), connect.NewRequest(&v1.HostVaultServiceRemoveLinksRequest{
		From: "Deltas.md", To: "Volga", Baseline: vault.readBaseline(t, "Deltas.md"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INTERNAL || res.Msg.GetDetail() != "no link to Volga is written here" {
		t.Errorf("RemoveLinks of a missing link returned %v, want INTERNAL with the detail naming name://Volga", res.Msg)
	}
}
