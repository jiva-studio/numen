package host_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

const staleDetail = "Deltas.md changed since it was read"

// linking is a host of a vault of rivers that serves the link edits and records the requests. Its
// AddLinks writes the first link, rejects the second as stale and fails the third with no code.
type linking struct {
	numenv1connect.UnimplementedHostVaultServiceHandler
	mu         sync.Mutex
	callError  v1.HostVaultErrorCode
	editError  v1.HostVaultErrorCode
	editDetail string
	added      *v1.HostVaultServiceAddLinksRequest
	updated    *v1.HostVaultServiceUpdateLinksRequest
	removed    *v1.HostVaultServiceRemoveLinksRequest
}

func (l *linking) AddLinks(
	_ context.Context, req *connect.Request[v1.HostVaultServiceAddLinksRequest],
) (*connect.Response[v1.HostVaultServiceAddLinksResponse], error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.added = req.Msg
	if l.callError != 0 {
		return connect.NewResponse(&v1.HostVaultServiceAddLinksResponse{Error: l.callError}), nil
	}
	return connect.NewResponse(&v1.HostVaultServiceAddLinksResponse{Results: []*v1.HostLinkAdded{
		{Stat: &v1.HostFileStat{Path: "Danube.md", Size: 64, ModTime: 2000, IsNote: true}},
		{Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_CONFLICT, Detail: staleDetail},
		{Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INTERNAL, Detail: diskFull},
	}}), nil
}

func (l *linking) UpdateLinks(
	_ context.Context, req *connect.Request[v1.HostVaultServiceUpdateLinksRequest],
) (*connect.Response[v1.HostVaultServiceUpdateLinksResponse], error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.updated = req.Msg
	if l.callError != 0 || l.editError != 0 {
		return connect.NewResponse(&v1.HostVaultServiceUpdateLinksResponse{Error: max(l.callError, l.editError), Detail: l.editDetail}), nil
	}
	return connect.NewResponse(&v1.HostVaultServiceUpdateLinksResponse{Stat: &v1.HostFileStat{Path: "Deltas.md", Size: 90, ModTime: 3000, IsNote: true}}), nil
}

func (l *linking) RemoveLinks(
	_ context.Context, req *connect.Request[v1.HostVaultServiceRemoveLinksRequest],
) (*connect.Response[v1.HostVaultServiceRemoveLinksResponse], error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.removed = req.Msg
	if l.callError != 0 || l.editError != 0 {
		return connect.NewResponse(&v1.HostVaultServiceRemoveLinksResponse{Error: max(l.callError, l.editError), Detail: l.editDetail}), nil
	}
	return connect.NewResponse(&v1.HostVaultServiceRemoveLinksResponse{Stat: &v1.HostFileStat{Path: "Deltas.md", Size: 40, ModTime: 4000, IsNote: true}}), nil
}

func linkingClient(t *testing.T, l *linking) *host.Client {
	t.Helper()
	server := httptest.NewServer(handler(l))
	t.Cleanup(server.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func getReadBaseline() domain.FileStat {
	return domain.FileStat{Path: "Deltas.md", Kind: domain.KindNote, Size: 48, ModTime: time.Unix(0, 1700)}
}

func TestTheClientSendsEachLinkAdditionWithItsBaselineAndRole(t *testing.T) {
	l := &linking{}

	_, err := linkingClient(t, l).AddLinks(t.Context(), []host.LinkAddition{
		{From: "Deltas.md", Baseline: getReadBaseline(), Link: domain.Link{
			Target: domain.Address{Scheme: domain.SchemeName, Value: "Danube"},
			Role:   domain.RoleParent, Type: "tributary", Label: "the Danube", Why: "it ends in the Danube delta",
		}},
		{From: "Deltas.md", Baseline: getReadBaseline(), Link: domain.Link{
			Target: domain.Address{Scheme: domain.SchemeNote, Value: "01JRIVERRHINE"}, Role: domain.RoleJump,
		}},
		{From: "Estuaries.md", Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Danube"}, Role: domain.RoleRef}},
	})
	if err != nil {
		t.Fatal(err)
	}

	links := l.added.GetLinks()
	if len(links) != 3 {
		t.Fatalf("the host received %d links, want 3: %v", len(links), l.added)
	}
	first := links[0]
	if first.GetFrom() != "Deltas.md" || first.GetTo() != "Danube" || first.GetRole() != v1.Role_ROLE_PARENT ||
		first.GetType() != "tributary" || first.GetLabel() != "the Danube" || first.GetNote() != "it ends in the Danube delta" {
		t.Errorf("the first link was sent to the host as %v", first)
	}
	if b := first.GetBaseline(); b.GetPath() != "Deltas.md" || b.GetSize() != 48 || b.GetModTime() != 1700 {
		t.Errorf("the baseline of the first link was sent to the host as %v, want Deltas.md of 48 bytes at 1700", b)
	}
	if second := links[1]; second.GetTo() != "note://01JRIVERRHINE" || second.GetRole() != v1.Role_ROLE_JUMP {
		t.Errorf("the second link was sent to the host as %v, want the target note://01JRIVERRHINE with role jump", second)
	}
	if third := links[2]; third.GetFrom() != "Estuaries.md" || third.GetRole() != v1.Role_ROLE_REF {
		t.Errorf("the third link was sent to the host as %v", third)
	}
}

func TestTheClientReturnsOneResultPerLinkAdditionInOrder(t *testing.T) {
	additions := []host.LinkAddition{
		{From: "Danube.md", Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Rhine"}, Role: domain.RoleJump}},
		{From: "Deltas.md", Baseline: getReadBaseline(), Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Danube"}, Role: domain.RoleParent}},
		{From: "Broken.md", Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Danube"}, Role: domain.RoleRef}},
	}

	results, err := linkingClient(t, &linking{}).AddLinks(t.Context(), additions)
	if err != nil {
		t.Fatalf("the additions failed as a whole: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("the results came back as %+v, want 3", results)
	}
	want := domain.FileStat{Path: "Danube.md", Kind: domain.KindNote, Size: 64, ModTime: time.Unix(0, 2000)}
	if r := results[0]; r.Err != nil || !r.Stat.ModTime.Equal(want.ModTime) || r.Stat.Path != want.Path || r.Stat.Size != want.Size {
		t.Errorf("the first result came back as %+v, want the stat %+v", r, want)
	}
	if r := results[1]; !errors.Is(r.Err, host.ErrStale) || !strings.Contains(r.Err.Error(), staleDetail) {
		t.Errorf("the second result came back as %+v, want ErrStale containing %q", r, staleDetail)
	}
	if r := results[2]; r.Err == nil || !strings.Contains(r.Err.Error(), diskFull) {
		t.Errorf("the third result came back as %+v, want an error containing %q", r, diskFull)
	}
}

func TestTheClientSendsTheChangeOfTheLinksToATarget(t *testing.T) {
	l := &linking{}

	stat, err := linkingClient(t, l).UpdateLinks(t.Context(), "Deltas.md", domain.Address{Scheme: domain.SchemeName, Value: "Danube"},
		domain.Link{Role: domain.RoleChild, Type: "distributary", Label: "Danube arm", Why: "the delta splits the river"}, getReadBaseline())
	if err != nil {
		t.Fatal(err)
	}

	r := l.updated
	if r.GetFrom() != "Deltas.md" || r.GetTo() != "Danube" || r.GetRole() != v1.Role_ROLE_CHILD || r.GetType() != "distributary" ||
		r.GetLabel() != "Danube arm" || r.GetNote() != "the delta splits the river" {
		t.Errorf("the update was sent to the host as %v", r)
	}
	if b := r.GetBaseline(); b.GetSize() != 48 || b.GetModTime() != 1700 {
		t.Errorf("the baseline was sent to the host as %v, want 48 bytes at 1700", b)
	}
	if stat.Path != "Deltas.md" || stat.Size != 90 || !stat.ModTime.Equal(time.Unix(0, 3000)) {
		t.Errorf("the update returned the stat %+v, want Deltas.md of 90 bytes at 3000", stat)
	}
}

func TestTheClientSendsTheRemovalOfTheLinksToATargetWithItsRole(t *testing.T) {
	for role, want := range map[domain.LinkRole]v1.Role{"": v1.Role_ROLE_UNSPECIFIED, domain.RoleParent: v1.Role_ROLE_PARENT} {
		l := &linking{}

		stat, err := linkingClient(t, l).RemoveLinks(t.Context(), "Deltas.md", domain.Address{Scheme: domain.SchemeNote, Value: "01JRIVERDANUBE"}, role, getReadBaseline())
		if err != nil {
			t.Fatal(err)
		}

		r := l.removed
		if r.GetFrom() != "Deltas.md" || r.GetTo() != "note://01JRIVERDANUBE" || r.GetRole() != want || r.GetBaseline().GetModTime() != 1700 {
			t.Errorf("the removal with role %q was sent to the host as %v, want role %v", role, r, want)
		}
		if stat.Size != 40 || !stat.ModTime.Equal(time.Unix(0, 4000)) {
			t.Errorf("the removal returned the stat %+v, want 40 bytes at 4000", stat)
		}
	}
}

func TestTheClientReturnsAStaleLinkEditAsErrStaleWithTheDetail(t *testing.T) {
	client := linkingClient(t, &linking{editError: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_CONFLICT, editDetail: staleDetail})
	target := domain.Address{Scheme: domain.SchemeName, Value: "Danube"}

	_, updateErr := client.UpdateLinks(t.Context(), "Deltas.md", target, domain.Link{Role: domain.RoleChild}, getReadBaseline())
	_, removeErr := client.RemoveLinks(t.Context(), "Deltas.md", target, "", getReadBaseline())

	for call, err := range map[string]error{"UpdateLinks": updateErr, "RemoveLinks": removeErr} {
		if !errors.Is(err, host.ErrStale) || !strings.Contains(err.Error(), staleDetail) {
			t.Errorf("%s returned %v, want ErrStale containing %q", call, err, staleDetail)
		}
	}
}

func TestTheClientKeepsTheDetailOfALinkEditThatFailed(t *testing.T) {
	const detail = "no link to Danube is written here"
	client := linkingClient(t, &linking{editError: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INTERNAL, editDetail: detail})
	target := domain.Address{Scheme: domain.SchemeName, Value: "Danube"}

	_, updateErr := client.UpdateLinks(t.Context(), "Deltas.md", target, domain.Link{Role: domain.RoleChild}, getReadBaseline())
	_, removeErr := client.RemoveLinks(t.Context(), "Deltas.md", target, "", getReadBaseline())

	for call, err := range map[string]error{"UpdateLinks": updateErr, "RemoveLinks": removeErr} {
		if err == nil || !strings.Contains(err.Error(), detail) {
			t.Errorf("%s returned %v, want an error containing %q", call, err, detail)
		}
	}
}

func TestTheClientReturnsNoVaultForEveryLinkEdit(t *testing.T) {
	client := linkingClient(t, &linking{callError: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_VAULT})
	target := domain.Address{Scheme: domain.SchemeName, Value: "Danube"}

	_, addErr := client.AddLinks(t.Context(), []host.LinkAddition{{From: "Deltas.md", Link: domain.Link{Target: target, Role: domain.RoleRef}}})
	_, updateErr := client.UpdateLinks(t.Context(), "Deltas.md", target, domain.Link{Role: domain.RoleChild}, getReadBaseline())
	_, removeErr := client.RemoveLinks(t.Context(), "Deltas.md", target, "", getReadBaseline())

	for call, err := range map[string]error{"AddLinks": addErr, "UpdateLinks": updateErr, "RemoveLinks": removeErr} {
		if !errors.Is(err, host.ErrNoVault) {
			t.Errorf("%s returned %v, want ErrNoVault", call, err)
		}
	}
}
