package host_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// relating is a host that serves the calls on connections and records what it was asked.
type relating struct {
	numenv1connect.UnimplementedHostVaultServiceHandler
	code    v1.HostVaultErrorCode
	saved   *v1.Connection
	joined  *v1.HostVaultServiceJoinNotesRequest
	created *v1.HostVaultServiceCreateRelatedNoteRequest
	removed []string
}

func (r *relating) GetConnection(
	_ context.Context, req *connect.Request[v1.HostVaultServiceGetConnectionRequest],
) (*connect.Response[v1.HostVaultServiceGetConnectionResponse], error) {
	return connect.NewResponse(&v1.HostVaultServiceGetConnectionResponse{
		Error: r.code,
		Connection: &v1.Connection{
			First: req.Msg.GetFirst(), Second: req.Msg.GetSecond(),
			FirstSide: &v1.ConnectionSide{
				Role: v1.Role_ROLE_PARENT, Type: req.Msg.GetType(), Note: "why", Label: "l", Marker: v1.Marker_MARKER_ARROW,
			},
		},
	}), nil
}

func (r *relating) SaveConnection(
	_ context.Context, req *connect.Request[v1.HostVaultServiceSaveConnectionRequest],
) (*connect.Response[v1.HostVaultServiceSaveConnectionResponse], error) {
	r.saved = req.Msg.GetConnection()
	return connect.NewResponse(&v1.HostVaultServiceSaveConnectionResponse{Error: r.code}), nil
}

func (r *relating) RemoveSide(
	_ context.Context, req *connect.Request[v1.HostVaultServiceRemoveSideRequest],
) (*connect.Response[v1.HostVaultServiceRemoveSideResponse], error) {
	r.removed = append(r.removed, req.Msg.GetPath()+">"+req.Msg.GetTo()+":"+req.Msg.GetType())
	return connect.NewResponse(&v1.HostVaultServiceRemoveSideResponse{Error: r.code}), nil
}

func (r *relating) RemoveConnection(
	_ context.Context, req *connect.Request[v1.HostVaultServiceRemoveConnectionRequest],
) (*connect.Response[v1.HostVaultServiceRemoveConnectionResponse], error) {
	r.removed = append(r.removed, req.Msg.GetFirst()+"<>"+req.Msg.GetSecond())
	return connect.NewResponse(&v1.HostVaultServiceRemoveConnectionResponse{Error: r.code}), nil
}

func (r *relating) JoinNotes(
	_ context.Context, req *connect.Request[v1.HostVaultServiceJoinNotesRequest],
) (*connect.Response[v1.HostVaultServiceJoinNotesResponse], error) {
	r.joined = req.Msg
	return connect.NewResponse(&v1.HostVaultServiceJoinNotesResponse{Error: r.code}), nil
}

func (r *relating) CreateRelatedNote(
	_ context.Context, req *connect.Request[v1.HostVaultServiceCreateRelatedNoteRequest],
) (*connect.Response[v1.HostVaultServiceCreateRelatedNoteResponse], error) {
	r.created = req.Msg
	return connect.NewResponse(&v1.HostVaultServiceCreateRelatedNoteResponse{
		Error: r.code, Path: "Untitled note.md",
	}), nil
}

func (r *relating) GetBacklinks(
	context.Context, *connect.Request[v1.GetBacklinksRequest],
) (*connect.Response[v1.GetBacklinksResponse], error) {
	return connect.NewResponse(&v1.GetBacklinksResponse{Links: []*v1.HostLink{{
		From: "idea.md", To: "area.md", TargetScheme: "name", TargetValue: "Area",
		Role: v1.Role_ROLE_PARENT, RoleFromHere: v1.Role_ROLE_CHILD, Marker: v1.Marker_MARKER_ARROW, Note: "n",
	}}}), nil
}

func relationsTo(t *testing.T, r *relating) host.Relations {
	t.Helper()
	server := httptest.NewServer(handler(r))
	t.Cleanup(server.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestAConnectionIsReadWithItsSides(t *testing.T) {
	got, err := relationsTo(t, &relating{}).GetConnection(t.Context(), "a.md", "b.md", "requires")

	want := host.Side{
		Role: host.RoleParent, Type: "requires", Why: "why", Label: "l", Marker: host.MarkerArrow,
	}
	if err != nil || got.A != "a.md" || got.B != "b.md" || got.AB == nil || *got.AB != want || got.BA != nil {
		t.Errorf("got %+v (%v)", got, err)
	}
}

func TestASavedConnectionCarriesBothSidesToTheHost(t *testing.T) {
	r := &relating{}

	err := relationsTo(t, r).SaveConnection(t.Context(), host.Connection{
		A: "a.md", B: "b.md",
		AB: &host.Side{Role: host.RoleJump, Marker: host.MarkerArrow},
		BA: &host.Side{Role: host.RoleJump, Label: "back"},
	})

	if err != nil {
		t.Fatal(err)
	}
	first, second := r.saved.GetFirstSide(), r.saved.GetSecondSide()
	if first.GetMarker() != v1.Marker_MARKER_ARROW || first.GetLabel() != "" || second.GetLabel() != "back" ||
		second.GetMarker() != v1.Marker_MARKER_UNSPECIFIED || second.GetRole() != v1.Role_ROLE_JUMP {
		t.Errorf("the host received %v", r.saved)
	}
}

func TestEachFailureOfAConnectionIsItsOwnError(t *testing.T) {
	cases := map[v1.HostVaultErrorCode]error{
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INVALID_CONNECTION: host.ErrInvalidConnection,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_HIERARCHY_CYCLE:    host.ErrHierarchyCycle,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NAME_EXHAUSTED:     host.ErrNoFreeName,
	}
	for code, want := range cases {
		err := relationsTo(t, &relating{code: code}).SaveConnection(t.Context(), host.Connection{A: "a", B: "b"})
		if !errors.Is(err, want) {
			t.Errorf("code %s: got %v, want %v", code, err, want)
		}
	}
	if !errors.Is(host.ErrHierarchyCycle, domain.ErrHierarchyCycle) {
		t.Error("the cycle error of the SDK is not the one of the notes library")
	}
}

func TestSidesAreRemovedAndJoinedAndNotesCreatedInARelation(t *testing.T) {
	r := &relating{}
	client := relationsTo(t, r)

	if err := client.RemoveSide(t.Context(), "a.md", "b.md", "x"); err != nil {
		t.Fatal(err)
	}
	if err := client.RemoveConnection(t.Context(), "a.md", "b.md", ""); err != nil {
		t.Fatal(err)
	}
	if err := client.Join(t.Context(), "a.md", "b.md", host.Side{Role: host.RoleChild, Label: "l"}); err != nil {
		t.Fatal(err)
	}
	path, err := client.CreateInRelation(t.Context(), host.RelatedNote{
		From: "a.md", Role: host.RoleParent, Type: "series",
	})

	if err != nil || path != "Untitled note.md" {
		t.Fatalf("path %q (%v)", path, err)
	}
	if len(r.removed) != 2 || r.removed[0] != "a.md>b.md:x" || r.removed[1] != "a.md<>b.md" {
		t.Errorf("removed %v", r.removed)
	}
	if r.joined.GetSide().GetRole() != v1.Role_ROLE_CHILD || r.joined.GetSide().GetLabel() != "l" {
		t.Errorf("joined %v", r.joined)
	}
	if r.created.GetRole() != v1.Role_ROLE_PARENT || r.created.GetType() != "series" || r.created.GetFrom() != "a.md" {
		t.Errorf("created %v", r.created)
	}
}

func TestABacklinkCarriesItsRoleFromTheNoteAskedAbout(t *testing.T) {
	got, err := relationsTo(t, &relating{}).ListBacklinks(t.Context(), "area.md")

	if err != nil || len(got) != 1 {
		t.Fatalf("got %+v (%v)", got, err)
	}
	b := got[0]
	if b.From != "idea.md" || b.Role != host.RoleParent || b.RoleFromHere != host.RoleChild ||
		b.Marker != host.MarkerArrow || b.Why != "n" {
		t.Errorf("backlink = %+v", b)
	}
}
