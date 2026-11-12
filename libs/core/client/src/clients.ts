/**
 * The service clients of the core and the protocol enum codes, for the applications.
 */
import { createClient, type Client, type Transport } from '@connectrpc/connect'
import {
  CommandErrorCode,
  CommandService,
  ErrorCode as ProtoErrorCode,
  FileService,
  FlushResult,
  Marker,
  Mode,
  NameSource,
  NoteService,
  Origin,
  PluginService,
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
import { keepaliveTransport, transport } from './transport'

export const createVaultClient = (customTransport: Transport = transport) =>
  createClient(VaultService, customTransport)
export const getNoteClient = (customTransport: Transport = transport) =>
  createClient(NoteService, customTransport)

export const vault = createVaultClient()
export const workspace = createClient(WorkspaceService, transport)
/** The workspace client for a request that must complete after the page unloads. */
export const keepaliveWorkspace = createClient(WorkspaceService, keepaliveTransport)
export const files = createClient(FileService, transport)
export const notes = getNoteClient()
export const search = createClient(SearchService, transport)
export const vaultsService = createClient(VaultsService, transport)
export const settingsService = createClient(SettingsService, transport)
export const windowService = createClient(WindowService, transport)
export const theme = createClient(ThemeService, transport)
export const texts = createClient(TextService, transport)
export const pluginCommands = createClient(CommandService, transport)
export const pluginService = createClient(PluginService, transport)

export const NAME_SOURCE = NameSource
export type NameSourceCode = NameSource

export const NOTE_MARKER = Marker
export type NoteMarkerCode = Marker

export const NOTE_ROLE = Role
export type NoteRoleCode = Role

export const NOTE_RELATION = Relation
export type NoteRelationCode = Relation

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
