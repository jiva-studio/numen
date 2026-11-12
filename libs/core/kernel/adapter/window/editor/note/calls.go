package note

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

// proseCalls are the calls the Prose part answers: it reads, writes, creates and renames notes.
type proseCalls interface {
	ReadNote(
		ctx context.Context, r *connect.Request[v1.ReadNoteRequest],
	) (*connect.Response[v1.ReadNoteResponse], error)
	WriteNote(
		ctx context.Context, r *connect.Request[v1.WriteNoteRequest],
	) (*connect.Response[v1.WriteNoteResponse], error)
	CreateNote(
		ctx context.Context, r *connect.Request[v1.CreateNoteRequest],
	) (*connect.Response[v1.CreateNoteResponse], error)
	RenameNote(
		ctx context.Context, r *connect.Request[v1.RenameNoteRequest],
	) (*connect.Response[v1.RenameNoteResponse], error)
}

// outlineCalls are the calls the Outline part answers: it answers the note a vault opens on, the neighborhood of a note and the headings of notes.
type outlineCalls interface {
	GetOpeningNote(
		ctx context.Context, _ *connect.Request[v1.GetOpeningNoteRequest],
	) (*connect.Response[v1.GetOpeningNoteResponse], error)
	GetNeighborhood(
		ctx context.Context, r *connect.Request[v1.GetNeighborhoodRequest],
	) (*connect.Response[v1.GetNeighborhoodResponse], error)
	ListHeadings(
		ctx context.Context,
		r *connect.Request[v1.ListHeadingsRequest],
	) (*connect.Response[v1.ListHeadingsResponse], error)
}

// linksCalls are the calls the Links part answers: it writes the links of notes and answers where links land and which notes link to a note.
type linksCalls interface {
	CreateRelatedNote(
		ctx context.Context, r *connect.Request[v1.CreateRelatedNoteRequest],
	) (*connect.Response[v1.CreateRelatedNoteResponse], error)
	ListBacklinks(
		ctx context.Context, r *connect.Request[v1.ListBacklinksRequest],
	) (*connect.Response[v1.ListBacklinksResponse], error)
	WriteLink(
		ctx context.Context, r *connect.Request[v1.WriteLinkRequest],
	) (*connect.Response[v1.WriteLinkResponse], error)
	ResolveAddresses(
		ctx context.Context, r *connect.Request[v1.ResolveAddressesRequest],
	) (*connect.Response[v1.ResolveAddressesResponse], error)
}

// connectionsCalls are the calls the Connections part answers: it reads and changes the relationship between two notes.
type connectionsCalls interface {
	GetConnection(
		ctx context.Context, r *connect.Request[v1.GetConnectionRequest],
	) (*connect.Response[v1.GetConnectionResponse], error)
	SaveConnection(
		ctx context.Context, r *connect.Request[v1.SaveConnectionRequest],
	) (*connect.Response[v1.SaveConnectionResponse], error)
	RemoveSide(
		ctx context.Context, r *connect.Request[v1.RemoveSideRequest],
	) (*connect.Response[v1.RemoveSideResponse], error)
	RemoveConnection(
		ctx context.Context, r *connect.Request[v1.RemoveConnectionRequest],
	) (*connect.Response[v1.RemoveConnectionResponse], error)
	JoinNotes(
		ctx context.Context, r *connect.Request[v1.JoinNotesRequest],
	) (*connect.Response[v1.JoinNotesResponse], error)
}
