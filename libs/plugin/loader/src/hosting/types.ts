/**
 * Types and interfaces for the plugin subsystem.
 */
import type {
  AgentHost,
  AgentProviderRegistration,
  AppInfo,
  BackgroundTaskHost,
  BufferHost,
  CommandItem,
  CommandRegistration,
  CommandsHost,
  Disposable,
  DropHost,
  DropRegistration,
  Caught,
  FilePlacement,
  FileSelectEvent,
  JsonValue,
  MountedView,
  NoticeHost,
  PluginContext,
  PluginLifecycle,
  ReaderRegistration,
  SettingsViewRegistration,
  SlotViewRegistration,
  SlotHost,
  CommandTarget,
  TabKindRegistration,
  TaskHost,
  VaultChangeEvent,
  VaultFiles,
  VaultNotes,
  VaultRelations,
  VaultTexts,
  VaultsHost,
  WindowHost,
  WorkspaceHost,
} from '@numen/plugin-sdk'
import type { AnyTabKind, SourceReader } from './tabKinds'
import type { DeclaredSlotEntry, SlotCommandLook } from '../slots/slotRegistry'
import type { WhenValues } from '../when/when'
import type { WhenReader } from '../when/whenReader'
import type { Component } from 'vue'
import type { DeclaredViewRegistration, MountAccess } from '../views'

export type {
  AgentProviderRegistration,
  CommandRegistration,
  Disposable,
  FileSelectEvent,
  PluginContext,
  PluginLifecycle,
  ReaderRegistration,
  SettingsViewRegistration,
  TabKindRegistration,
  TaskHost,
  VaultChangeEvent,
}

/** One command as a plugin manifest declares it under `contributes.commands`, with the parts its plugin has. */
export interface CommandDeclaration {
  readonly pluginId: string
  readonly id: string
  readonly title: string
  readonly icon?: string
  readonly when?: string
}

/** One handler of a shared command as a plugin manifest declares it under `contributes.handlers`. */
export interface HandlerDeclaration {
  readonly pluginId: string
  readonly command: string
  readonly title: string
  readonly when?: string
  readonly hasItems: boolean
}

/**
 * One run of a handler in its plugin's backend: over the target, with `itemId` the id of the item
 * picked and `values` the values of the arguments the shared command declares, by name.
 */
export interface BackendHandlerRun {
  readonly pluginId: string
  readonly command: string
  readonly target: CommandTarget
  readonly itemId?: string | undefined
  readonly values?: Readonly<Record<string, string>> | undefined
}

/** What a run of a command is given: the id of the item picked, and the values of the arguments of a shared command by name. */
export interface CommandRun {
  readonly itemId?: string | undefined
  readonly values?: Readonly<Record<string, string>> | undefined
  readonly isAwaited?: boolean | undefined
}

export interface PluginPaletteCommand {
  readonly id: string
  readonly title: string
  readonly when?: string
  readonly icon?: string
}

/** What a slot's command finder reads of the commands the plugins registered with the host. */
export interface HostCommandReads {
  readonly getPluginCommands: () => readonly PluginPaletteCommand[]
  readonly hasPluginItems: (command: string) => boolean
}

export type SlotCommandFinder = (
  command: string,
  target: CommandTarget | undefined,
) => SlotCommandLook | undefined

export interface PluginHostDeps {
  readonly getContentTypes?: () => Readonly<Record<string, string>>
  readonly getWindowValues?: () => WhenValues
  readonly whenReader?: WhenReader
  readonly getCommandDeclarations?: () => readonly CommandDeclaration[] | undefined
  readonly getHandlerDeclarations?: () => readonly HandlerDeclaration[] | undefined
  readonly listBackendHandlerItems?: (
    pluginId: string,
    command: string,
    target: CommandTarget,
  ) => Promise<readonly CommandItem[]>
  readonly runBackendHandler?: (run: BackendHandlerRun) => Promise<void>
  readonly hasPluginFrontend?: (pluginId: string) => boolean
  readonly hasPluginBackend?: (pluginId: string) => boolean
  readonly isPluginFailed?: (pluginId: string) => boolean
  readonly waitForPlugin?: (pluginId: string) => Promise<void>
  readonly reportCommandIssue?: (pluginId: string, message: string) => void
  readonly reportViewIssue?: (pluginId: string, message: string) => void
  readonly registerKind: (kind: AnyTabKind) => () => void
  readonly registerExtensionReader: (
    extension: string,
    readItem: SourceReader,
    icon?: Component,
  ) => () => void
  readonly getReaderExtensions?: (pluginId: string, readerId: string) => readonly string[]
  readonly registerEditor?: (
    type: string,
    open: (
      path: string,
      title?: string,
      options?: FilePlacement | { readonly where?: FilePlacement | undefined },
    ) => void,
  ) => (() => void) | void
  readonly openTab: (kind: string, at?: string) => Promise<string>
  readonly openTabAdjacent?: (kind: string, at?: string) => Promise<string>
  readonly navigateTab?: (kind: string, path: string) => Promise<string>
  readonly closeTab?: (id: string) => void
  readonly getTabState?: (id: string) => object | undefined
  readonly getSetting?: (pluginId: string, key: string) => JsonValue | undefined
  readonly onSettingChange?: (
    pluginId: string,
    key: string,
    handler: (value: JsonValue | undefined) => void | Promise<void>,
  ) => () => void
  readonly closeTabsForPlugin?: (pluginId: string) => void
  readonly onError?: (pluginId: string, error: Caught) => void

  readonly readNote?:
    ((path: string) => Promise<{ body?: string; error?: string } | string | undefined>) | undefined
  readonly writeNote?: ((path: string, body: string) => Promise<void>) | undefined
  readonly getWorkspace?: () => WorkspaceHost | undefined
  readonly getNotes?: () => VaultNotes | undefined
  readonly getRelations?: () => VaultRelations | undefined
  readonly getFiles?: () => VaultFiles | undefined
  readonly getAgent?: () => AgentHost | undefined
  readonly getCommands?: () => CommandsHost | undefined
  readonly getVaults?: () => VaultsHost | undefined
  readonly getWindow?: () => WindowHost | undefined
  readonly getBuffers?: () => BufferHost | undefined
  readonly getNoteBuffer?: () => NoteBuffer | undefined
  readonly resolveLink?: (from: string, address: string) => Promise<string>
  readonly getTexts?: (pluginId: string) => VaultTexts
  readonly getNotices?: (pluginId: string) => NoticeHost
  readonly getTasks?: (pluginId: string) => TaskHost | undefined
  readonly getSlotEntries?: () => readonly DeclaredSlotEntry[] | undefined
  readonly createSlotCommandFinder?: (own: HostCommandReads) => SlotCommandFinder
  readonly getSlotWhenValues?: (target: CommandTarget | undefined) => WhenValues
  readonly reportSlotIssue?: (
    pluginId: string,
    message: string,
    severity: 'warning' | 'error',
  ) => void
  readonly getBackgroundTasks?: () => BackgroundTaskHost | undefined
  readonly clearNotices?: (pluginId: string) => void
  readonly app?: AppInfo
  readonly openExternal?: (url: string) => Promise<void>
}

/**
 * The window's store of open notes, by the id of the tab each is open in: it writes after typing
 * settles, against the file the tab read, and holds the text when the file changed on disk.
 */
export interface NoteBuffer {
  open(id: string, path: string): void
  reload(id: string, path: string): void
  close(id: string): Promise<boolean>
  setBody(id: string, body: string): void
  save(id: string): void
  /** Writes the text on screen over the file that changed on disk. */
  keep(id: string): void
  /** Reads the file that changed on disk, dropping the text on screen. */
  take(id: string): void
  flushPending(id: string): Promise<void>
  getOpenNote(id: string): {
    readonly path: string
    readonly body: string
    readonly state: string
  }
  getErrorMessage(id: string): string
}

/** What a plugin registers into the window, and the views it mounts; the window keeps what is registered. */
export interface RegistrationPort {
  readonly registerReader: (pluginId: string, registration: ReaderRegistration) => Disposable
  readonly registerSettingsView: (
    pluginId: string,
    registration: SettingsViewRegistration,
  ) => Disposable
  readonly registerSlotView: (
    pluginId: string,
    slot: string,
    registration: SlotViewRegistration,
  ) => Disposable
  readonly registerTabKind: (pluginId: string, registration: TabKindRegistration) => Disposable
  readonly registerCommand: (pluginId: string, registration: CommandRegistration) => Disposable
  readonly registerAgentProvider: (
    pluginId: string,
    registration: AgentProviderRegistration,
  ) => Disposable
  readonly registerView: (
    pluginId: string,
    registration: DeclaredViewRegistration,
    permissions?: readonly string[],
  ) => Disposable
  readonly registerDropHandler: (pluginId: string, registration: DropRegistration) => Disposable
  readonly mountView: (
    capability: string,
    element: HTMLElement,
    context: object,
    access: MountAccess,
  ) => MountedView
}

/** The settings of the plugins as the window holds them. */
export interface SettingsPort {
  readonly getSetting: <T extends JsonValue = JsonValue>(
    pluginId: string,
    key: string,
  ) => T | undefined
  readonly onSettingChange: <T extends JsonValue = JsonValue>(
    pluginId: string,
    key: string,
    handler: (value: T) => void | Promise<void>,
  ) => Disposable
}

/** The events of the vault and the file selection, which plugins listen to and raise. */
export interface EventPort {
  readonly onVaultChange: (
    pluginId: string,
    handler: (event: VaultChangeEvent) => void | Promise<void>,
  ) => Disposable
  readonly onFileSelect: (
    pluginId: string,
    handler: (event: FileSelectEvent) => void | Promise<void>,
  ) => Disposable
  readonly emitVaultChange: (pluginId: string, event: VaultChangeEvent) => void
  readonly emitFileSelect: (pluginId: string, event: FileSelectEvent) => void
}

/** The tabs of the window. A window without tabs answers no id. */
export interface TabsPort {
  readonly openTab: (kind: string, at?: string) => Promise<string> | undefined
  readonly navigateTab: (kind: string, path: string) => Promise<string> | undefined
  readonly closeTab: (id: string) => void
}

/** What the window offers of the vault; a service the window lacks answers undefined. */
export interface VaultPort {
  readonly getNotes: () => VaultNotes | undefined
  readonly getRelations: () => VaultRelations | undefined
  readonly getFiles: () => VaultFiles | undefined
  readonly getTexts: (pluginId: string) => VaultTexts | undefined
  readonly getBuffers: () => BufferHost | undefined
  readonly getVaults: () => VaultsHost | undefined
}

/** What the window offers of its interface; a service the window lacks answers undefined. */
export interface UiPort {
  readonly getWorkspace: () => WorkspaceHost | undefined
  readonly getWindow: () => WindowHost | undefined
  readonly getCommands: () => CommandsHost | undefined
  readonly getSlots: () => SlotHost
  readonly getNotices: (pluginId: string) => NoticeHost
  readonly drops: DropHost
}

/** The agent and the background work of the window; a service the window lacks answers undefined. */
export interface WorkPort {
  readonly getAgent: () => AgentHost | undefined
  readonly getTasks: (pluginId: string) => TaskHost | undefined
  readonly getBackgroundTasks: () => BackgroundTaskHost | undefined
}

/** What a window gives `createPluginContext` to build the context of a plugin, in groups of services. */
export interface PluginContextHost {
  readonly app: AppInfo
  readonly target: string
  readonly openExternal: (url: string) => Promise<void>
  readonly registrations: RegistrationPort
  readonly settings: SettingsPort
  readonly events: EventPort
  readonly tabs: TabsPort
  readonly vault: VaultPort
  readonly ui: UiPort
  readonly work: WorkPort
}

/** What the host runs with: the deps of the window, with the one reader of its conditions. */
export type HostDeps = PluginHostDeps & { readonly whenReader: WhenReader }
