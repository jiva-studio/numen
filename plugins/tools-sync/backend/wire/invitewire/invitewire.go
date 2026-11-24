// Package invitewire answers the methods of the settings view that share a folder by a link and receive one.
package invitewire

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/shareinvite"
)

// Invites answers the methods about sharing a folder by a link, as the three parts below.
type Invites struct {
	*Links
	*Requests
	*Joins
}

// Links answers the methods about the links this computer made.
type Links struct {
	start  *shareinvite.Start
	list   *shareinvite.ListInvitations
	revoke *shareinvite.Revoke
}

// Requests answers the methods about the people who wait for the sharer to answer.
type Requests struct {
	list    *shareinvite.ListRequests
	approve *shareinvite.Approve
	decline *shareinvite.Decline
}

// Joins answers the methods about the links this computer pasted.
type Joins struct {
	read   *shareinvite.ReadLink
	join   *shareinvite.Join
	status *shareinvite.GetStatus
}

// New returns the methods over the dependencies.
func New(deps shareinvite.Deps) *Invites {
	return &Invites{
		Links: &Links{
			start: shareinvite.NewStart(deps), list: shareinvite.NewListInvitations(deps), revoke: shareinvite.NewRevoke(deps),
		},
		Requests: &Requests{
			list: shareinvite.NewListRequests(deps), approve: shareinvite.NewApprove(deps), decline: shareinvite.NewDecline(deps),
		},
		Joins: &Joins{read: shareinvite.NewReadLink(deps), join: shareinvite.NewJoin(deps), status: shareinvite.NewGetStatus(deps)},
	}
}

func formatTime(at time.Time) string { return at.UTC().Format(time.RFC3339) }

// StartShareInvitation makes the link a folder is shared by.
func (s *Links) StartShareInvitation(ctx context.Context, request api.StartShareInvitationRequest) (api.StartShareInvitationResponse, error) {
	started, err := s.start.Execute(ctx, domain.VaultID(request.VaultID), request.Folder)
	if err != nil {
		return api.StartShareInvitationResponse{}, err
	}
	return api.StartShareInvitationResponse{ID: started.ID, Link: started.Link, ExpiresAt: formatTime(started.ExpiresAt)}, nil
}

// ListShareInvitations lists the links that work still.
func (s *Links) ListShareInvitations(ctx context.Context, _ api.ListShareInvitationsRequest) (api.ListShareInvitationsResponse, error) {
	listed := s.list.Execute(ctx)
	invitations := make([]api.ShareInvitation, len(listed))
	for i, one := range listed {
		invitations[i] = api.ShareInvitation{ID: one.ID, Folder: one.Folder, ExpiresAt: formatTime(one.ExpiresAt), IsRequested: one.IsRequested}
	}
	return api.ListShareInvitationsResponse{Invitations: invitations}, nil
}

// RevokeShareInvitation makes a link stop working.
func (s *Links) RevokeShareInvitation(ctx context.Context, request api.RevokeShareInvitationRequest) (api.RevokeShareInvitationResponse, error) {
	return api.RevokeShareInvitationResponse{}, s.revoke.Execute(ctx, request.ID)
}

// ListShareRequests lists the devices that wait for an answer.
func (s *Requests) ListShareRequests(ctx context.Context, _ api.ListShareRequestsRequest) (api.ListShareRequestsResponse, error) {
	listed := s.list.Execute(ctx)
	requests := make([]api.ShareRequest, len(listed))
	for i, one := range listed {
		requests[i] = api.ShareRequest{
			RequestID: one.ID, VaultID: string(one.VaultID), Folder: one.Folder, DeviceName: one.DeviceName, Fingerprint: one.Fingerprint, Code: one.Code,
		}
	}
	return api.ListShareRequestsResponse{Requests: requests}, nil
}

// ApproveShareRequest lets the device read the folder.
func (s *Requests) ApproveShareRequest(ctx context.Context, request api.ApproveShareRequestRequest) (api.ApproveShareRequestResponse, error) {
	return api.ApproveShareRequestResponse{}, s.approve.Execute(ctx, request.RequestID)
}

// DeclineShareRequest refuses the device.
func (s *Requests) DeclineShareRequest(ctx context.Context, request api.DeclineShareRequestRequest) (api.DeclineShareRequestResponse, error) {
	return api.DeclineShareRequestResponse{}, s.decline.Execute(ctx, request.RequestID)
}

// GetShareInvitationPreview reads a link without contacting anybody.
func (s *Joins) GetShareInvitationPreview(_ context.Context, request api.PreviewShareInvitationRequest) (api.PreviewShareInvitationResponse, error) {
	preview, err := s.read.Execute(request.Link)
	if err != nil {
		return api.PreviewShareInvitationResponse{}, err
	}
	return api.PreviewShareInvitationResponse{
		Folder: preview.Folder, OwnerFingerprint: preview.OwnerFingerprint, ExpiresAt: formatTime(preview.ExpiresAt), IsReadOnly: preview.IsReadOnly,
	}, nil
}

// AcceptShareInvitation sends the owner of a link a request to read the folder.
func (s *Joins) AcceptShareInvitation(ctx context.Context, request api.AcceptShareInvitationRequest) (api.AcceptShareInvitationResponse, error) {
	joined, err := s.join.Execute(ctx, request.Link, request.DeviceName)
	if err != nil {
		return api.AcceptShareInvitationResponse{}, err
	}
	return api.AcceptShareInvitationResponse{RequestID: joined.ID, Folder: joined.Folder, OwnerFingerprint: joined.OwnerFingerprint}, nil
}

// GetShareInvitationStatus says how a request of this computer stands.
func (s *Joins) GetShareInvitationStatus(_ context.Context, request api.GetShareInvitationStatusRequest) (api.GetShareInvitationStatusResponse, error) {
	status, err := s.status.Execute(request.RequestID)
	if err != nil {
		return api.GetShareInvitationStatusResponse{}, err
	}
	return api.GetShareInvitationStatusResponse{
		Phase: api.ShareRequestPhase(status.Phase), Code: status.Code, Folder: status.Folder, OwnerFingerprint: status.OwnerFingerprint,
		ExpiresAt: formatTime(status.ExpiresAt), Error: status.Error,
	}, nil
}
