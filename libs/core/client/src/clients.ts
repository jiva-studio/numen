/**
 * The factories of the service clients of the core and the protocol enum codes, for the
 * applications.
 */
import { createClient, type Client, type Transport } from '@connectrpc/connect'
import {
  CatalogService,
  CommandErrorCode,
  CommandService,
  ConnectionCallService,
  ContributionService,
  ErrorCode as ProtoErrorCode,
  FileService,
  FolderDialogService,
  FlushResult,
  IdentityService,
  LicenseService,
  LinkDirection,
  LogService,
  Marker,
  Mode,
  NameSource,
  NoteService,
  Origin,
  PluginDataService,
  PluginHealthService,
  PluginProcessService,
  PluginService,
  ProviderConnectionService,
  Relation,
  Role,
  SearchMode,
  SearchService,
  SettingsService,
  TaskKind,
  TabMoveErrorCode,
  TextErrorCode,
  TextService,
  TextVersionService,
  ThemeService,
  Unit,
  VaultService,
  VaultsErrorCode,
  VaultsService,
  WindowService,
  WorkspaceService,
} from '@numen/protocol'
/** The clients of the core's services, each sending its requests over the transport it is given. */
export const createVaultRpc = (transport: Transport) => createClient(VaultService, transport)
export const createNoteRpc = (transport: Transport) => createClient(NoteService, transport)
export const createWorkspaceRpc = (transport: Transport) =>
  createClient(WorkspaceService, transport)
export const createFileRpc = (transport: Transport) => createClient(FileService, transport)
export const createSearchRpc = (transport: Transport) => createClient(SearchService, transport)
export const createVaultsRpc = (transport: Transport) => createClient(VaultsService, transport)
export const createFolderDialogRpc = (transport: Transport) =>
  createClient(FolderDialogService, transport)
export const createSettingsRpc = (transport: Transport) => createClient(SettingsService, transport)
export const createWindowRpc = (transport: Transport) => createClient(WindowService, transport)
export const createThemeRpc = (transport: Transport) => createClient(ThemeService, transport)
export const createTextRpc = (transport: Transport) => createClient(TextService, transport)
export const createCommandRpc = (transport: Transport) => createClient(CommandService, transport)
export const createPluginRpc = (transport: Transport) => createClient(PluginService, transport)
export const createPluginProcessRpc = (transport: Transport) =>
  createClient(PluginProcessService, transport)
export const createPluginDataRpc = (transport: Transport) =>
  createClient(PluginDataService, transport)
export const createContributionRpc = (transport: Transport) =>
  createClient(ContributionService, transport)
export const createTextVersionRpc = (transport: Transport) =>
  createClient(TextVersionService, transport)
export const createPluginHealthRpc = (transport: Transport) =>
  createClient(PluginHealthService, transport)
export const createCatalogRpc = (transport: Transport) => createClient(CatalogService, transport)
export const createProviderConnectionRpc = (transport: Transport) =>
  createClient(ProviderConnectionService, transport)
export const createConnectionCallRpc = (transport: Transport) =>
  createClient(ConnectionCallService, transport)
export const createLogRpc = (transport: Transport) => createClient(LogService, transport)
export const createLicenseRpc = (transport: Transport) => createClient(LicenseService, transport)
export const createIdentityRpc = (transport: Transport) => createClient(IdentityService, transport)

export const NAME_SOURCE = NameSource
export type NameSourceCode = NameSource

export const NOTE_MARKER = Marker
export type NoteMarkerCode = Marker

export const NOTE_ROLE = Role
export type NoteRoleCode = Role

export const NOTE_RELATION = Relation
export type NoteRelationCode = Relation

export const NOTE_LINK_DIRECTION = LinkDirection
export type NoteLinkDirectionCode = LinkDirection

export const FLUSH_RESULT = FlushResult
export type FlushResultCode = FlushResult

export const SEARCH_MODE = SearchMode
export type SearchModeCode = SearchMode

export const TASK_KIND = TaskKind
export const TASK_UNIT = Unit
export type TaskUnitCode = Unit

export const VAULTS_ERROR_CODE = VaultsErrorCode
export type VaultsErrorCodeValue = VaultsErrorCode

export const COMMAND_ERROR_CODE = CommandErrorCode

export const TEXT_ERROR_CODE = TextErrorCode

export const TAB_MOVE_ERROR_CODE = TabMoveErrorCode
export type TabMoveErrorCodeValue = TabMoveErrorCode

export const THEME_MODE = Mode
export type ThemeModeCode = Mode

export const THEME_ORIGIN = Origin
export type ThemeOriginCode = Origin

export const NOTE_ERROR_CODE = ProtoErrorCode
export type NoteErrorCodeValue = ProtoErrorCode

/** The calls of TextService the texts are read and edited through. */
export type TextClient = Pick<Client<typeof TextService>, 'readLiveText' | 'createTextEdit'>
