/**
 * Core ConnectRPC clients and protocol enum mappings for application hosts.
 */
import { createClient, type Client, type Transport } from '@connectrpc/connect'
import {
  AgentService,
  CAPABILITY_AGENT_PROVIDER,
  CommandErrorCode,
  CommandService,
  CompletionConsent,
  ErrorCode as ProtoErrorCode,
  FileService,
  FlushResult,
  Mode,
  NameSource,
  NoteService,
  Origin,
  PluginService,
  Presence,
  Relation,
  Role,
  SearchMode,
  SearchService,
  SettingsService,
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
  type TextSegment as TextSegmentMessage,
  type TextVersion as TextVersionMessage,
} from '@numen/protocol'
import type { LiveText, Selector, TextSegment, TextVersion, VaultTexts } from './texts'
import { transport } from './transport'

export const createVaultClient = (customTransport: Transport = transport) =>
  createClient(VaultService, customTransport)
export const createAgentClient = (customTransport: Transport = transport) =>
  createClient(AgentService, customTransport)
export const createWorkspaceClient = (customTransport: Transport = transport) =>
  createClient(WorkspaceService, customTransport)
export const createFileClient = (customTransport: Transport = transport) =>
  createClient(FileService, customTransport)
export const createNoteClient = (customTransport: Transport = transport) =>
  createClient(NoteService, customTransport)
export const createSearchClient = (customTransport: Transport = transport) =>
  createClient(SearchService, customTransport)
export const createVaultsClient = (customTransport: Transport = transport) =>
  createClient(VaultsService, customTransport)
export const createSettingsClient = (customTransport: Transport = transport) =>
  createClient(SettingsService, customTransport)
export const createWindowClient = (customTransport: Transport = transport) =>
  createClient(WindowService, customTransport)
export const createThemeClient = (customTransport: Transport = transport) =>
  createClient(ThemeService, customTransport)
export const createTextVersionClient = (customTransport: Transport = transport) =>
  createClient(TextVersionService, customTransport)
export const createTextClient = (customTransport: Transport = transport) =>
  createClient(TextService, customTransport)
export const createCommandClient = (customTransport: Transport = transport) =>
  createClient(CommandService, customTransport)
export const createPluginClient = (customTransport: Transport = transport) =>
  createClient(PluginService, customTransport)

export const vault = createVaultClient()
export const agentService = createAgentClient()
export const workspace = createWorkspaceClient()
export const files = createFileClient()
export const notes = createNoteClient()
export const search = createSearchClient()
export const vaultsService = createVaultsClient()
export const settingsService = createSettingsClient()
export const windowService = createWindowClient()
export const theme = createThemeClient()
export const textVersions = createTextVersionClient()
export const texts = createTextClient()
export const pluginCommands = createCommandClient()
export const pluginService = createPluginClient()

export const NAME_SOURCE = NameSource
export type NameSourceCode = NameSource

export const NOTE_ROLE = Role
export type NoteRoleCode = Role

export const NOTE_RELATION = Relation
export type NoteRelationCode = Relation

export const COMPLETION_CONSENT = CompletionConsent
export type CompletionConsentCode = CompletionConsent

export const FLUSH_RESULT = FlushResult
export type FlushResultCode = FlushResult

export const MODEL_PRESENCE = Presence
export type ModelPresenceCode = Presence

export const SEARCH_MODE = SearchMode
export type SearchModeCode = SearchMode

export const TASK_UNIT = Unit
export type TaskUnitCode = Unit

export const VAULTS_ERROR_CODE = VaultsErrorCode
export type VaultsErrorCodeValue = VaultsErrorCode

export const COMMAND_ERROR_CODE = CommandErrorCode
export type CommandErrorCodeValue = CommandErrorCode

export const TEXT_ERROR_CODE = TextErrorCode
export type TextErrorCodeValue = TextErrorCode

export const THEME_MODE = Mode
export type ThemeModeCode = Mode

export const THEME_ORIGIN = Origin
export type ThemeOriginCode = Origin

export const NOTE_ERROR_CODE = ProtoErrorCode
export type NoteErrorCodeValue = ProtoErrorCode

export const AGENT_PROVIDER_CAPABILITY = CAPABILITY_AGENT_PROVIDER

/** The calls of TextService the texts are read and edited through. */
export type TextClient = Pick<Client<typeof TextService>, 'readLiveText' | 'createTextEdit'>

const readSelector = (raw: string): Selector | undefined => {
  if (raw === '') return undefined
  try {
    const parsed: unknown = JSON.parse(raw)
    return typeof parsed === 'object' && parsed !== null ? (parsed as Selector) : undefined
  } catch {
    return undefined
  }
}

const readSegment = (one: TextSegmentMessage): TextSegment => {
  const selector = readSelector(one.selector)
  return {
    text: one.text,
    ...(one.heading === '' ? {} : { heading: one.heading }),
    ...(selector === undefined ? {} : { selector }),
  }
}

const readVersion = (one: TextVersionMessage): TextVersion => ({
  id: one.id,
  path: one.path,
  pluginId: one.pluginId,
  key: one.key,
  sourceVersionId: one.sourceVersionId,
  kind: one.kind,
  file: one.file,
  createTime: new Date(Number(one.createTimeUnixMs)),
})

const writeSegment = (one: TextSegment) => ({
  text: one.text,
  heading: one.heading ?? '',
  selector: one.selector === undefined ? '' : JSON.stringify(one.selector),
})

/** The texts a plugin's views read, and edit in the plugin's folder. */
export const createPluginTexts = (client: TextClient, pluginId: string): VaultTexts => ({
  readLiveText: async (path): Promise<LiveText | null> => {
    const answer = await client.readLiveText({ path })
    if (answer.version === undefined) return null
    return { version: readVersion(answer.version), segments: answer.segments.map(readSegment) }
  },
  createEdit: async (path, sourceVersionId, segments) => {
    const answer = await client.createTextEdit({
      pluginId,
      path,
      sourceVersionId,
      segments: segments.map(writeSegment),
    })
    if (answer.version !== undefined && answer.error === TextErrorCode.UNSPECIFIED) {
      return { ok: true, value: readVersion(answer.version) }
    }
    return { ok: false, error: answer.error === TextErrorCode.MISSING ? 'missing' : 'invalid' }
  },
})
