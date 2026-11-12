package vault

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/reflect/protoreflect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// The calls of HostVaultService are answered by the groups below, each with its own state, and
// NewHandler serves one procedure of the service from the group that answers it.
type (
	filesCalls struct {
		*calls
	}
	watchCalls       struct{ *calls }
	cacheCalls       struct{ *calls }
	lookupsCalls     struct{ *calls }
	arrangeCalls     struct{ *calls }
	connectionsCalls struct{ *calls }
	vaultsCalls      struct{ *calls }
)

// router serves the procedures of one service from the handler of each.
type router struct {
	routes  map[string]http.Handler
	methods protoreflect.MethodDescriptors
	options []connect.HandlerOption
}

func addUnary[Q, R any](r *router, procedure, method string, call func(context.Context, *connect.Request[Q]) (*connect.Response[R], error)) {
	r.routes[procedure] = connect.NewUnaryHandler(procedure, call,
		connect.WithSchema(r.methods.ByName(protoreflect.Name(method))), connect.WithHandlerOptions(r.options...))
}

func addStream[Q, R any](r *router, procedure, method string, call func(context.Context, *connect.Request[Q], *connect.ServerStream[R]) error) {
	r.routes[procedure] = connect.NewServerStreamHandler(procedure, call,
		connect.WithSchema(r.methods.ByName(protoreflect.Name(method))), connect.WithHandlerOptions(r.options...))
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if handler, ok := r.routes[req.URL.Path]; ok {
		handler.ServeHTTP(w, req)
		return
	}
	http.NotFound(w, req)
}

// NewHandler serves HostVaultService for the plugins that call the host, over the vault the source lends.
func NewHandler(auth Authority, source Source, options ...connect.HandlerOption) (string, http.Handler) {
	base := &calls{auth: auth, source: source}
	r := &router{
		routes:  map[string]http.Handler{},
		methods: v1.File_numen_v1_hostvault_proto.Services().ByName("HostVaultService").Methods(),
		options: options,
	}
	files := &filesCalls{base}
	watch, cache := &watchCalls{base}, &cacheCalls{base}
	lookups, arrange := &lookupsCalls{base}, &arrangeCalls{base}
	connections, vaults := &connectionsCalls{base}, &vaultsCalls{base}
	addUnary(r, numenv1connect.HostVaultServiceGetHostVaultProcedure, "GetHostVault", files.GetHostVault)
	addUnary(r, numenv1connect.HostVaultServiceListHostVaultsProcedure, "ListHostVaults", files.ListHostVaults)
	addUnary(r, numenv1connect.HostVaultServiceEnsureVaultIndexedProcedure, "EnsureVaultIndexed", files.EnsureVaultIndexed)
	addStream(r, numenv1connect.HostVaultServiceWatchMovesProcedure, "WatchMoves", watch.WatchMoves)
	addUnary(r, numenv1connect.HostVaultServiceStatFileProcedure, "StatFile", files.StatFile)
	addUnary(r, numenv1connect.HostVaultServiceReadFileProcedure, "ReadFile", files.ReadFile)
	addUnary(r, numenv1connect.HostVaultServiceWriteFileProcedure, "WriteFile", files.WriteFile)
	addUnary(r, numenv1connect.HostVaultServiceCreateFileProcedure, "CreateFile", files.CreateFile)
	addUnary(r, numenv1connect.HostVaultServiceLockVaultProcedure, "LockVault", files.LockVault)
	addUnary(r, numenv1connect.HostVaultServiceUnlockVaultProcedure, "UnlockVault", files.UnlockVault)
	addUnary(r, numenv1connect.HostVaultServiceScanFilesProcedure, "ScanFiles", files.ScanFiles)
	addUnary(r, numenv1connect.HostVaultServiceGetNoteRefsProcedure, "GetNoteRefs", lookups.GetNoteRefs)
	addUnary(r, numenv1connect.HostVaultServiceGetNoteTypesProcedure, "GetNoteTypes", lookups.GetNoteTypes)
	addUnary(r, numenv1connect.HostVaultServiceGetNamedPathsProcedure, "GetNamedPaths", lookups.GetNamedPaths)
	addUnary(r, numenv1connect.HostVaultServiceGetPathsOfTypeProcedure, "GetPathsOfType", lookups.GetPathsOfType)
	addUnary(r, numenv1connect.HostVaultServiceGetLinksProcedure, "GetLinks", lookups.GetLinks)
	addUnary(r, numenv1connect.HostVaultServiceGetBacklinksProcedure, "GetBacklinks", lookups.GetBacklinks)
	addUnary(r, numenv1connect.HostVaultServiceResolveLinksProcedure, "ResolveLinks", lookups.ResolveLinks)
	addUnary(r, numenv1connect.HostVaultServiceGetParseProblemsProcedure, "GetParseProblems", arrange.GetParseProblems)
	addUnary(r, numenv1connect.HostVaultServiceReadCacheProcedure, "ReadCache", cache.ReadCache)
	addUnary(r, numenv1connect.HostVaultServiceWriteCacheProcedure, "WriteCache", cache.WriteCache)
	addStream(r, numenv1connect.HostVaultServiceWatchVaultProcedure, "WatchVault", watch.WatchVault)
	addUnary(r, numenv1connect.HostVaultServiceAddLinksProcedure, "AddLinks", arrange.AddLinks)
	addUnary(r, numenv1connect.HostVaultServiceUpdateLinksProcedure, "UpdateLinks", arrange.UpdateLinks)
	addUnary(r, numenv1connect.HostVaultServiceRemoveLinksProcedure, "RemoveLinks", arrange.RemoveLinks)
	addUnary(r, numenv1connect.HostVaultServiceGetConnectionProcedure, "GetConnection", connections.GetConnection)
	addUnary(r, numenv1connect.HostVaultServiceSaveConnectionProcedure, "SaveConnection", connections.SaveConnection)
	addUnary(r, numenv1connect.HostVaultServiceRemoveSideProcedure, "RemoveSide", connections.RemoveSide)
	addUnary(r, numenv1connect.HostVaultServiceRemoveConnectionProcedure, "RemoveConnection", connections.RemoveConnection)
	addUnary(r, numenv1connect.HostVaultServiceJoinNotesProcedure, "JoinNotes", connections.JoinNotes)
	addUnary(r, numenv1connect.HostVaultServiceCreateRelatedNoteProcedure, "CreateRelatedNote", connections.CreateRelatedNote)
	addUnary(r, numenv1connect.HostVaultServiceGetVaultProblemsProcedure, "GetVaultProblems", arrange.GetVaultProblems)
	addUnary(r, numenv1connect.HostVaultServiceRenameNoteProcedure, "RenameNote", arrange.RenameNote)
	addUnary(r, numenv1connect.HostVaultServiceMoveNotesProcedure, "MoveNotes", arrange.MoveNotes)
	addUnary(r, numenv1connect.HostVaultServiceRemoveNotesProcedure, "RemoveNotes", arrange.RemoveNotes)
	addUnary(r, numenv1connect.HostVaultServiceAddVaultProcedure, "AddVault", vaults.AddVault)
	addUnary(r, numenv1connect.HostVaultServiceRenameVaultProcedure, "RenameVault", vaults.RenameVault)
	addUnary(r, numenv1connect.HostVaultServiceRemoveVaultProcedure, "RemoveVault", vaults.RemoveVault)
	addUnary(r, numenv1connect.HostVaultServiceOpenVaultProcedure, "OpenVault", vaults.OpenVault)
	return "/" + numenv1connect.HostVaultServiceName + "/", r
}
