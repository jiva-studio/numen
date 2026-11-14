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
  ErrorCode as ProtoErrorCode,
  FileService,
  FlushResult,
  LinkDirection,
  Marker,
  Mode,
  NameSource,
  NoteService,
  Origin,
  PluginHealthService,
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
  ThemeService,
  Unit,
  VaultService,
  VaultsErrorCode,
  VaultsService,
  WindowService,
  WorkspaceService,
} from '@numen/protocol'
/** The clients of the core's services, each sending its requests over the transport it is given. */
export const createVaultClient = (transport: Transport) => createClient(VaultService, transport)
export const createNoteClient = (transport: Transport) => createClient(NoteService, transport)
export const createWorkspaceClient = (transport: Transport) =>
  createClient(WorkspaceService, transport)
export const createFileClient = (transport: Transport) => createClient(FileService, transport)
export const createSearchClient = (transport: Transport) => createClient(SearchService, transport)
export const createVaultsClient = (transport: Transport) => createClient(VaultsService, transport)
export const createSettingsClient = (transport: Transport) =>
  createClient(SettingsService, transport)
export const createWindowClient = (transport: Transport) => createClient(WindowService, transport)
export const createThemeServiceClient = (transport: Transport) =>
  createClient(ThemeService, transport)
export const createTextClient = (transport: Transport) => createClient(TextService, transport)
export const createCommandClient = (transport: Transport) => createClient(CommandService, transport)
export const createPluginClient = (transport: Transport) => createClient(PluginService, transport)
export const createPluginHealthClient = (transport: Transport) =>
  createClient(PluginHealthService, transport)
export const createCatalogClient = (transport: Transport) => createClient(CatalogService, transport)
export const createProviderConnectionClient = (transport: Transport) =>
  createClient(ProviderConnectionService, transport)
export const createConnectionCallServiceClient = (transport: Transport) =>
  createClient(ConnectionCallService, transport)

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
