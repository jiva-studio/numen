package vault

import (
	"context"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/reflect/protoreflect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// The calls of the host vault services are answered by the groups below, each with its own state, and
// Mount serves one procedure of a service from the group that answers it.
type (
	filesCalls struct {
		*calls
	}
	watchCalls   struct{ *calls }
	cacheCalls   struct{ *calls }
	lookupsCalls struct{ *calls }
	arrangeCalls struct{ *calls }
	vaultsCalls  struct{ *calls }
)

// router serves the procedures of one service from the handler of each.
type router struct {
	name    string
	routes  map[string]http.Handler
	methods protoreflect.MethodDescriptors
	options []connect.HandlerOption
}

func newRouter(file protoreflect.FileDescriptor, service string, options []connect.HandlerOption) *router {
	return &router{
		name:    service,
		routes:  map[string]http.Handler{},
		methods: file.Services().ByName(protoreflect.Name(strings.TrimPrefix(service, "numen.v1."))).Methods(),
		options: options,
	}
}

func (r *router) mount(mux *http.ServeMux) {
	mux.Handle("/"+r.name+"/", r)
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

// Mount serves the vault services for the plugins that call the host, over the vault the source lends: one
// router for each service, each answering its procedures from the group that holds them.
func Mount(mux *http.ServeMux, auth Authority, source Source, options ...connect.HandlerOption) {
	base := &calls{auth: auth, source: source}
	files := &filesCalls{base}
	watch, cache := &watchCalls{base}, &cacheCalls{base}
	lookups, arrange := &lookupsCalls{base}, &arrangeCalls{base}
	vaults := &vaultsCalls{base}

	vault := newRouter(v1.File_numen_v1_hostvault_proto, numenv1connect.HostVaultServiceName, options)
	addUnary(vault, numenv1connect.HostVaultServiceGetHostVaultProcedure, "GetHostVault", files.GetHostVault)
	addUnary(vault, numenv1connect.HostVaultServiceEnsureVaultIndexedProcedure, "EnsureVaultIndexed", files.EnsureVaultIndexed)
	addStream(vault, numenv1connect.HostVaultServiceWatchMovesProcedure, "WatchMoves", watch.WatchMoves)
	addUnary(vault, numenv1connect.HostVaultServiceLockVaultProcedure, "LockVault", files.LockVault)
	addUnary(vault, numenv1connect.HostVaultServiceUnlockVaultProcedure, "UnlockVault", files.UnlockVault)
	addUnary(vault, numenv1connect.HostVaultServiceGetParseProblemsProcedure, "GetParseProblems", arrange.GetParseProblems)
	addUnary(vault, numenv1connect.HostVaultServiceGetVaultProblemsProcedure, "GetVaultProblems", arrange.GetVaultProblems)
	vault.mount(mux)

	list := newRouter(v1.File_numen_v1_vaults_proto, numenv1connect.VaultsServiceName, options)
	addUnary(list, numenv1connect.VaultsServiceListVaultsProcedure, "ListVaults", vaults.ListVaults)
	addUnary(list, numenv1connect.VaultsServiceAddVaultProcedure, "AddVault", vaults.AddVault)
	addUnary(list, numenv1connect.VaultsServiceRenameVaultProcedure, "RenameVault", vaults.RenameVault)
	addUnary(list, numenv1connect.VaultsServiceRemoveVaultProcedure, "RemoveVault", vaults.RemoveVault)
	addUnary(list, numenv1connect.VaultsServiceOpenVaultProcedure, "OpenVault", vaults.OpenVault)
	list.mount(mux)

	mountDocuments(mux, options, files, lookups, arrange, cache)
}

func mountDocuments(
	mux *http.ServeMux, options []connect.HandlerOption,
	files *filesCalls, lookups *lookupsCalls, arrange *arrangeCalls, cache *cacheCalls,
) {
	file := newRouter(v1.File_numen_v1_hostfile_proto, numenv1connect.HostFileServiceName, options)
	addUnary(file, numenv1connect.HostFileServiceStatFileProcedure, "StatFile", files.StatFile)
	addUnary(file, numenv1connect.HostFileServiceReadFileProcedure, "ReadFile", files.ReadFile)
	addUnary(file, numenv1connect.HostFileServiceWriteFileProcedure, "WriteFile", files.WriteFile)
	addUnary(file, numenv1connect.HostFileServiceCreateFileProcedure, "CreateFile", files.CreateFile)
	addUnary(file, numenv1connect.HostFileServiceScanFilesProcedure, "ScanFiles", files.ScanFiles)
	addUnary(file, numenv1connect.HostFileServiceListFileChangesProcedure, "ListFileChanges", lookups.ListFileChanges)
	addUnary(file, numenv1connect.HostFileServiceMoveFileProcedure, "MoveFile", files.MoveFile)
	addUnary(file, numenv1connect.HostFileServiceRemoveFileProcedure, "RemoveFile", files.RemoveFile)
	addUnary(file, numenv1connect.HostFileServiceReadFileRangeProcedure, "ReadFileRange", files.ReadFileRange)
	addUnary(file, numenv1connect.HostFileServicePlaceFileProcedure, "PlaceFile", files.PlaceFile)
	file.mount(mux)

	note := newRouter(v1.File_numen_v1_hostnote_proto, numenv1connect.HostNoteServiceName, options)
	addUnary(note, numenv1connect.HostNoteServiceGetNoteRefsProcedure, "GetNoteRefs", lookups.GetNoteRefs)
	addUnary(note, numenv1connect.HostNoteServiceGetNoteTypesProcedure, "GetNoteTypes", lookups.GetNoteTypes)
	addUnary(note, numenv1connect.HostNoteServiceGetNamedPathsProcedure, "GetNamedPaths", lookups.GetNamedPaths)
	addUnary(note, numenv1connect.HostNoteServiceGetPathsOfTypeProcedure, "GetPathsOfType", lookups.GetPathsOfType)
	addUnary(note, numenv1connect.HostNoteServiceReadFrontmatterProcedure, "ReadFrontmatter", lookups.ReadFrontmatter)
	addUnary(note, numenv1connect.HostNoteServiceWriteFrontmatterProcedure, "WriteFrontmatter", arrange.WriteFrontmatter)
	addUnary(note, numenv1connect.HostNoteServiceRenameNoteProcedure, "RenameNote", arrange.RenameNote)
	addUnary(note, numenv1connect.HostNoteServiceMoveNotesProcedure, "MoveNotes", arrange.MoveNotes)
	addUnary(note, numenv1connect.HostNoteServiceRemoveNotesProcedure, "RemoveNotes", arrange.RemoveNotes)
	note.mount(mux)

	link := newRouter(v1.File_numen_v1_hostlink_proto, numenv1connect.HostLinkServiceName, options)
	addUnary(link, numenv1connect.HostLinkServiceGetLinksProcedure, "GetLinks", lookups.GetLinks)
	addUnary(link, numenv1connect.HostLinkServiceGetBacklinksProcedure, "GetBacklinks", lookups.GetBacklinks)
	addUnary(link, numenv1connect.HostLinkServiceResolveLinksProcedure, "ResolveLinks", lookups.ResolveLinks)
	addUnary(link, numenv1connect.HostLinkServiceAddLinksProcedure, "AddLinks", arrange.AddLinks)
	addUnary(link, numenv1connect.HostLinkServiceUpdateLinksProcedure, "UpdateLinks", arrange.UpdateLinks)
	addUnary(link, numenv1connect.HostLinkServiceRemoveLinksProcedure, "RemoveLinks", arrange.RemoveLinks)
	link.mount(mux)

	cacheRouter := newRouter(v1.File_numen_v1_hostcache_proto, numenv1connect.HostCacheServiceName, options)
	addUnary(cacheRouter, numenv1connect.HostCacheServiceReadCacheProcedure, "ReadCache", cache.ReadCache)
	addUnary(cacheRouter, numenv1connect.HostCacheServiceWriteCacheProcedure, "WriteCache", cache.WriteCache)
	cacheRouter.mount(mux)
}
