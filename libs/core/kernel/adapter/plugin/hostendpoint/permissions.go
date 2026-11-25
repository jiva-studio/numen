package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// noPermission marks a procedure any running plugin may call. The handler of such a procedure grants nothing
// beyond the plugin itself: its own store, its own cache, its own health, or a grant it checks itself.
const noPermission = ""

func getProcedurePermissions() map[string]string {
	permissions := getHostPermissions()
	maps.Copy(permissions, getIndexPermissions())
	maps.Copy(permissions, getVaultPermissions())
	return permissions
}

func getHostPermissions() map[string]string {
	return map[string]string{
		numenv1connect.HostDependencyServiceFetchModelFileProcedure: noPermission,
		numenv1connect.ConnectionCallServiceCallProcedure:           noPermission,
		numenv1connect.ConnectionCallServiceCancelCallProcedure:     noPermission,
		numenv1connect.ConnectionCallServiceListRowsProcedure:       noPermission,
		numenv1connect.ConnectionCallServiceListRowOptionsProcedure: noPermission,
		numenv1connect.HostStoreServiceReadStoreFileProcedure:       noPermission,
		numenv1connect.HostStoreServiceAppendStoreFileProcedure:     noPermission,
		numenv1connect.HostStoreServiceListStoreFilesProcedure:      noPermission,
		numenv1connect.HostStoreServiceWriteStoreFileProcedure:      noPermission,
		numenv1connect.HostStoreServiceDeleteStoreFileProcedure:     noPermission,
		numenv1connect.HostStoreServiceRenameStoreFileProcedure:     noPermission,
		numenv1connect.HostQueueServiceEnqueueWorkProcedure:         noPermission,
		numenv1connect.HostQueueServiceReportTaskProcedure:          noPermission,
		numenv1connect.HostQueueServiceFinishTaskProcedure:          noPermission,
		numenv1connect.PluginHealthServiceReportHealthProcedure:     noPermission,
		numenv1connect.PluginHealthServiceClearHealthProcedure:      noPermission,
		numenv1connect.HostProcessServiceReleaseProcessProcedure:    noPermission,
		numenv1connect.LogServiceWriteLogProcedure:                  noPermission,
		numenv1connect.HostCacheServiceReadCacheProcedure:           noPermission,
		numenv1connect.HostCacheServiceWriteCacheProcedure:          noPermission,
		numenv1connect.HostAccountServiceSignAsDeviceProcedure:      manifest.PermissionIdentity,
		numenv1connect.HostAccountServiceGetPublicIdentityProcedure: manifest.PermissionIdentity,
		numenv1connect.HostAccountServiceSealDataProcedure:          manifest.PermissionIdentity,
		numenv1connect.HostAccountServiceOpenDataProcedure:          manifest.PermissionIdentity,
		numenv1connect.HostAccountServiceOpenSealedBoxProcedure:     manifest.PermissionIdentity,
		numenv1connect.HostAccountServiceGetPendingRestoreProcedure: manifest.PermissionIdentityLink,
		numenv1connect.HostAccountServiceSignAsRecoveryProcedure:    manifest.PermissionIdentityLink,
		numenv1connect.HostAccountServiceCompleteRestoreProcedure:   manifest.PermissionIdentityLink,
		numenv1connect.HostAccountServiceGetLinkWorkProcedure:       manifest.PermissionIdentityLink,
		numenv1connect.HostAccountServiceSetLinkChannelProcedure:    manifest.PermissionIdentityLink,
		numenv1connect.HostAccountServicePutLinkInboundProcedure:    manifest.PermissionIdentityLink,
		numenv1connect.HostAccountServiceGetChainPublishProcedure:   manifest.PermissionIdentityLink,
		numenv1connect.HostAccountServicePutChainPublishedProcedure: manifest.PermissionIdentityLink,
		numenv1connect.HostAccountServiceAdoptChainProcedure:        manifest.PermissionIdentityLink,
	}
}

func getIndexPermissions() map[string]string {
	return map[string]string{
		numenv1connect.HostTextServiceListTextVersionsProcedure:      manifest.PermissionVaultRead,
		numenv1connect.HostTextServiceReadTextProcedure:              manifest.PermissionVaultRead,
		numenv1connect.HostTextServicePutTextVersionProcedure:        manifest.PermissionVaultWrite,
		numenv1connect.HostSearchServiceGetEmbeddingStateProcedure:   manifest.PermissionIndexRead,
		numenv1connect.HostSearchServiceGetIndexStatsProcedure:       manifest.PermissionIndexRead,
		numenv1connect.HostSearchServiceEmbedTextsProcedure:          manifest.PermissionIndexRead,
		numenv1connect.HostSearchServiceSampleSimilaritiesProcedure:  manifest.PermissionIndexRead,
		numenv1connect.HostSearchServiceGetBestSimilarityProcedure:   manifest.PermissionIndexRead,
		numenv1connect.HostSearchServiceFindNearestChunksProcedure:   manifest.PermissionIndexRead,
		numenv1connect.HostSearchServiceSearchTextProcedure:          manifest.PermissionIndexRead,
		numenv1connect.HostSearchServiceStartSearchProcedure:         manifest.PermissionIndexRead,
		numenv1connect.HostSearchServiceGetSearchProcedure:           manifest.PermissionIndexRead,
		numenv1connect.HostSearchServiceDeleteSearchProcedure:        manifest.PermissionIndexRead,
		numenv1connect.HostSearchServiceListSearchFunctionsProcedure: manifest.PermissionIndexRead,
		numenv1connect.HostSearchServiceGetSearchFloorProcedure:      manifest.PermissionSearchPolicy,
		numenv1connect.HostSearchServiceSetSearchFloorProcedure:      manifest.PermissionSearchPolicy,
		numenv1connect.PluginToolServiceListToolsProcedure:           manifest.PermissionVaultRead,
		numenv1connect.PluginToolServiceCallToolProcedure:            manifest.PermissionVaultRead,
		numenv1connect.HostWindowServiceFocusNoteProcedure:           manifest.PermissionWindowFocus,
		numenv1connect.HostWindowServiceFocusFileProcedure:           manifest.PermissionWindowFocus,
		numenv1connect.HostWindowServiceReadOpenTabsProcedure:        manifest.PermissionWindowFocus,
		numenv1connect.HostWindowServiceActivateTabProcedure:         manifest.PermissionWindowFocus,
	}
}

func getVaultPermissions() map[string]string {
	return map[string]string{
		numenv1connect.HostVaultServiceGetHostVaultProcedure:      manifest.PermissionVaultRead,
		numenv1connect.HostVaultServiceIndexVaultProcedure:        manifest.PermissionVaultRead,
		numenv1connect.HostVaultServiceListParseProblemsProcedure: manifest.PermissionVaultRead,
		numenv1connect.HostVaultServiceListVaultProblemsProcedure: manifest.PermissionVaultRead,
		numenv1connect.HostVaultServiceWatchIndexChangesProcedure: manifest.PermissionVaultRead,
		numenv1connect.HostVaultServiceListAbsentFilesProcedure:   manifest.PermissionVaultRead,
		numenv1connect.HostVaultServiceLockVaultProcedure:         manifest.PermissionVaultWrite,
		numenv1connect.HostVaultServiceUnlockVaultProcedure:       manifest.PermissionVaultWrite,
		numenv1connect.HostVaultServicePutAbsentFilesProcedure:    manifest.PermissionVaultWrite,
		numenv1connect.HostVaultServiceFillFileProcedure:          manifest.PermissionVaultWrite,
		numenv1connect.HostVaultServiceEvictFileProcedure:         manifest.PermissionVaultWrite,
		numenv1connect.VaultsServiceListVaultsProcedure:           manifest.PermissionVaultRead,
		numenv1connect.VaultsServiceAddVaultProcedure:             manifest.PermissionVaultManage,
		numenv1connect.VaultsServiceRenameVaultProcedure:          manifest.PermissionVaultManage,
		numenv1connect.VaultsServiceRemoveVaultProcedure:          manifest.PermissionVaultManage,
		numenv1connect.VaultsServiceOpenVaultProcedure:            manifest.PermissionVaultManage,
		numenv1connect.HostFileServiceStatFileProcedure:           manifest.PermissionVaultRead,
		numenv1connect.HostFileServiceReadFileProcedure:           manifest.PermissionVaultRead,
		numenv1connect.HostFileServiceListFileChangesProcedure:    manifest.PermissionVaultRead,
		numenv1connect.HostFileServiceReadFileRangeProcedure:      manifest.PermissionVaultRead,
		numenv1connect.HostFileServiceWriteFileProcedure:          manifest.PermissionVaultWrite,
		numenv1connect.HostFileServiceCreateFileProcedure:         manifest.PermissionVaultWrite,
		numenv1connect.HostFileServiceScanFilesProcedure:          manifest.PermissionVaultWrite,
		numenv1connect.HostFileServiceMoveFileProcedure:           manifest.PermissionVaultWrite,
		numenv1connect.HostFileServiceRemoveFileProcedure:         manifest.PermissionVaultWrite,
		numenv1connect.HostFileServicePlaceFileProcedure:          manifest.PermissionVaultWrite,
		numenv1connect.HostNoteServiceListNoteRefsProcedure:       manifest.PermissionVaultRead,
		numenv1connect.HostNoteServiceListNoteTypesProcedure:      manifest.PermissionVaultRead,
		numenv1connect.HostNoteServiceListNamedPathsProcedure:     manifest.PermissionVaultRead,
		numenv1connect.HostNoteServiceListPathsOfTypeProcedure:    manifest.PermissionVaultRead,
		numenv1connect.HostNoteServiceReadFrontmatterProcedure:    manifest.PermissionVaultRead,
		numenv1connect.HostNoteServiceWriteFrontmatterProcedure:   manifest.PermissionVaultWrite,
		numenv1connect.HostNoteServiceRenameNoteProcedure:         manifest.PermissionVaultWrite,
		numenv1connect.HostNoteServiceMoveNotesProcedure:          manifest.PermissionVaultWrite,
		numenv1connect.HostNoteServiceRemoveNotesProcedure:        manifest.PermissionVaultWrite,
		numenv1connect.HostLinkServiceListLinksProcedure:          manifest.PermissionVaultRead,
		numenv1connect.HostLinkServiceListBacklinksProcedure:      manifest.PermissionVaultRead,
		numenv1connect.HostLinkServiceResolveLinksProcedure:       manifest.PermissionVaultRead,
		numenv1connect.HostLinkServiceAddLinksProcedure:           manifest.PermissionVaultWrite,
		numenv1connect.HostLinkServiceUpdateLinksProcedure:        manifest.PermissionVaultWrite,
		numenv1connect.HostLinkServiceRemoveLinksProcedure:        manifest.PermissionVaultWrite,
	}
}

var errProcedureRefused = errors.New("no permission grants this procedure to a plugin")

// permissionGate refuses, before a handler runs, a call whose token names no running plugin, whose procedure
// has no entry in getProcedurePermissions, or whose plugin does not hold the permission of the procedure.
type permissionGate struct {
	endpoint    *Endpoint
	permissions map[string]string
}

func (e *Endpoint) newPermissionOption() connect.HandlerOption {
	return connect.WithInterceptors(permissionGate{endpoint: e, permissions: getProcedurePermissions()})
}

func (g permissionGate) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if err := g.check(req.Spec().Procedure, req.Header()); err != nil {
			return nil, err
		}
		return next(ctx, req)
	}
}

func (g permissionGate) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (g permissionGate) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if err := g.check(conn.Spec().Procedure, conn.RequestHeader()); err != nil {
			return err
		}
		return next(ctx, conn)
	}
}

func (g permissionGate) check(procedure string, header http.Header) error {
	pluginID, err := g.endpoint.Authenticate(header.Get("Authorization"))
	if err != nil {
		return err
	}
	permission, isListed := g.permissions[procedure]
	if !isListed {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("plugin %s calls %s: %w", pluginID, procedure, errProcedureRefused))
	}
	if permission == noPermission {
		return nil
	}
	if err := g.endpoint.callers.Activation.CheckPluginPermission(pluginID, permission); err != nil {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("plugin %s: %w", pluginID, err))
	}
	return nil
}
