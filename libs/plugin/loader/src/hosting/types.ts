/**
 * Types and interfaces for the plugin subsystem.
 */
import type {
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
  HealthHost,
  JsonValue,
  MountedView,
  NoticeHost,
  ProcessHost,
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
  ConnectionHost,
  VaultsHost,
  WindowHost,
  WorkspaceHost,
} from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import type { ContributionMatch } from '../loading'
import type { AnyTabKind, SourceReader } from './tabKinds'
import type { DeclaredSlotEntry, SlotCommandLook } from '../slots'
import type { WhenValues, WhenReader } from '../when'
import type { HealthReports } from '../pluginHealth'
import type { Component } from 'vue'
import type { DeclaredViewRegistration, MountAccess } from '../views'

/** One command as a plugin manifest declares it under `contributes.commands`, with the parts its plugin has. */
export interface CommandDeclaration {
  readonly pluginId: string
  readonly id: string
  readonly title: string
  readonly icon?: string
  readonly when?: string
  readonly permission?: string
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

/** Where a note opens: the pane it opens in, and the line it opens on. */
export interface EditorPlacement {
  readonly where?: FilePlacement | undefined
  readonly line?: number | undefined
}

/**
 * The window the plugins run in: its target, which the plugins it activates read as theirs, its
 * application, how it opens web links and reports the failure of a plugin, and its workspace,
 * commands, frame and notices.
 */
export interface HostWindow {
  readonly target: string
  readonly app?: AppInfo
  readonly openExternal?: (url: string) => Promise<void>
  readonly onError?: (pluginId: string, error: Caught) => void
  readonly getWorkspace?: () => WorkspaceHost | undefined
  readonly getCommands?: () => CommandsHost | undefined
  readonly getWindow?: () => WindowHost | undefined
  readonly getNotices?: (pluginId: string) => NoticeHost
  readonly clearNotices?: (pluginId: string) => void
}

/**
 * The plugins of the window: the parts each has, whether its start failed, the starts and the
 * settings. startPlugin settles once the plugin is active or has failed, and does nothing for a
 * plugin that is off; findPluginsOf matches the manifests read so far.
 */
export interface HostPlugins {
  readonly hasPluginFrontend?: (pluginId: string) => boolean
  readonly hasPluginBackend?: (pluginId: string) => boolean
  readonly isPluginFailed?: (pluginId: string) => boolean
  readonly startPlugin?: (pluginId: string) => Promise<void>
  readonly startPluginsOf?: (match: ContributionMatch) => Promise<void>
  readonly findPluginsOf?: (match: ContributionMatch) => readonly string[]
  readonly getSetting?: (pluginId: string, key: string) => JsonValue | undefined
  readonly onSettingChange?: (
    pluginId: string,
    key: string,
    handler: (value: JsonValue | undefined) => void | Promise<void>,
  ) => () => void
}

/** The commands the plugins declare, the handlers their backends run, and what their conditions read. */
export interface HostCommands {
  readonly getCommandDeclarations?: () => readonly CommandDeclaration[] | undefined
  readonly getHandlerDeclarations?: () => readonly HandlerDeclaration[] | undefined
  readonly listBackendHandlerItems?: (
    pluginId: string,
    command: string,
    target: CommandTarget,
  ) => Promise<readonly CommandItem[]>
  readonly runBackendHandler?: (run: BackendHandlerRun) => Promise<void>
  readonly reportCommandIssue?: (pluginId: string, message: string) => void
  readonly getContentTypes?: () => Readonly<Record<string, string>>
  readonly getWindowValues?: () => WhenValues
  readonly whenReader?: WhenReader
}

/**
 * The slots of the window and the views drawn in them. getSlotEntries reads the slot entries from the
 * manifests of the plugins whose context the host made, keyed by plugin id.
 */
export interface HostSlots {
  readonly getSlotEntries?: (
    manifests: ReadonlyMap<string, Manifest>,
  ) => readonly DeclaredSlotEntry[] | undefined
  readonly createSlotCommandFinder?: (own: HostCommandReads) => SlotCommandFinder
  readonly getSlotWhenValues?: (target: CommandTarget | undefined) => WhenValues
  readonly reportSlotIssue?: (
    pluginId: string,
    message: string,
    severity: 'warning' | 'error',
  ) => void
  readonly reportViewIssue?: (pluginId: string, message: string) => void
}

/**
 * The tab kinds and readers the plugins bring into the window, the tabs of a plugin that stops, and
 * the notes the tab kinds of the plugins show.
 */
export interface HostKinds {
  readonly registerKind: (kind: AnyTabKind) => () => void
  readonly registerExtensionReader?: (
    extension: string,
    readItem: SourceReader,
    icon?: Component,
  ) => () => void
  readonly getReaderExtensions?: (pluginId: string, readerId: string) => readonly string[]
  readonly registerEditor?: (
    type: string,
    open: (path: string, title?: string, options?: FilePlacement | EditorPlacement | null) => void,
  ) => (() => void) | void
  readonly closeTabsForPlugin?: (pluginId: string) => void
  readonly getNoteBuffer?: () => NoteBuffer | undefined
  readonly readNote?:
    ((path: string) => Promise<{ body?: string; error?: string } | string | undefined>) | undefined
  readonly writeNote?: ((path: string, body: string) => Promise<void>) | undefined
}

/** The tabs of the window. A window that opens no tab for a plugin leaves out openTab. */
export interface HostTabs {
  readonly openTab?: (kind: string, at?: string) => Promise<string>
  readonly openTabAdjacent?: (kind: string, at?: string) => Promise<string>
  readonly navigateTab?: (kind: string, path: string) => Promise<string>
  readonly closeTab?: (id: string) => void
  readonly getTabState?: (id: string) => object | undefined
}

/** The vault services of the window, and how it resolves a link between notes. */
export interface HostVault {
  readonly getNotes?: () => VaultNotes | undefined
  readonly getRelations?: () => VaultRelations | undefined
  readonly getFiles?: () => VaultFiles | undefined
  readonly getVaults?: () => VaultsHost | undefined
  readonly getBuffers?: () => BufferHost | undefined
  readonly getTexts?: (pluginId: string) => VaultTexts
  readonly resolveLink?: (from: string, address: string) => Promise<string>
}

/** The background work of the window: the tasks, the connections, the processes and the health items. */
export interface HostWork {
  readonly getTasks?: (pluginId: string) => TaskHost | undefined
  readonly getConnections?: (pluginId: string) => ConnectionHost
  readonly getBackgroundTasks?: () => BackgroundTaskHost | undefined
  readonly getProcesses?: () => ProcessHost | undefined
  readonly health?: HealthReports
}

/** What the plugin host is made from, by the responsibility of each part. */
export interface PluginHostDeps {
  readonly window: HostWindow
  readonly plugins?: HostPlugins
  readonly commands?: HostCommands
  readonly slots?: HostSlots
  readonly kinds: HostKinds
  readonly tabs?: HostTabs
  readonly vault?: HostVault
  readonly work?: HostWork
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

/** The registrations a plugin makes into the registry of the host. */
export interface ContextRegistrationHost {
  readonly registerReader: (pluginId: string, registration: ReaderRegistration) => Disposable
  readonly registerSettingsView: (
    pluginId: string,
    registration: SettingsViewRegistration,
  ) => Disposable
  readonly registerTabKind: (pluginId: string, registration: TabKindRegistration) => Disposable
  readonly registerCommand: (pluginId: string, registration: CommandRegistration) => Disposable
}

/** The views and drop handlers a plugin registers, and the views it mounts. */
export interface ContextViewHost {
  readonly registerSlotView: (
    pluginId: string,
    slot: string,
    registration: SlotViewRegistration,
  ) => Disposable
  readonly registerView: (
    pluginId: string,
    registration: DeclaredViewRegistration,
    permissions?: readonly string[],
  ) => Disposable
  mountView(
    capability: string,
    element: HTMLElement,
    context: object,
    access: MountAccess,
  ): MountedView
  readonly registerDropHandler: (pluginId: string, registration: DropRegistration) => Disposable
  readonly drops: DropHost
}

/** The settings a plugin reads and follows. */
export interface ContextSettingHost {
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

/** The events of the vault and of the selection that plugins listen to and emit. */
export interface ContextEventHost {
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

/** The tabs a plugin opens, navigates and closes. A window without tabs answers no id. */
export interface ContextTabHost {
  readonly openTab: (kind: string, at?: string) => Promise<string> | undefined
  readonly navigateTab: (kind: string, path: string) => Promise<string> | undefined
  readonly closeTab: (id: string) => void
}

/** The services that read and write the vault; a service the window lacks answers undefined. */
export interface ContextVaultHost {
  readonly getNotes: () => VaultNotes | undefined
  readonly getRelations: () => VaultRelations | undefined
  readonly getFiles: () => VaultFiles | undefined
  readonly getBuffers: () => BufferHost | undefined
  readonly getVaults: () => VaultsHost | undefined
  readonly getTexts: (pluginId: string) => VaultTexts | undefined
}

/**
 * The window a plugin draws in: its application, its target, how it opens web links, its workspace,
 * commands, frame, notices and slots. A service the window lacks answers undefined.
 */
export interface ContextUiHost {
  readonly app: AppInfo
  readonly target: string
  readonly openExternal: (url: string) => Promise<void>
  readonly getWorkspace: () => WorkspaceHost | undefined
  readonly getCommands: () => CommandsHost | undefined
  readonly isCommandListed: (command: string) => boolean
  readonly getCommandPermission?: (command: string) => string | undefined
  readonly getWindow: () => WindowHost | undefined
  readonly getNotices: (pluginId: string) => NoticeHost
  readonly getSlots: () => SlotHost
}

/** The background work of the window and the health items of a plugin. A service the window lacks answers undefined. */
export interface ContextWorkHost {
  readonly getTasks: (pluginId: string) => TaskHost | undefined
  readonly getConnections: (pluginId: string) => ConnectionHost | undefined
  readonly getBackgroundTasks: () => BackgroundTaskHost | undefined
  readonly getProcesses: () => ProcessHost | undefined
  readonly getHealth: (pluginId: string) => HealthHost | undefined
}

/** What a plugin context is made from, by the responsibility of each part. */
export interface PluginContextHost {
  readonly registration: ContextRegistrationHost
  readonly views: ContextViewHost
  readonly settings: ContextSettingHost
  readonly events: ContextEventHost
  readonly tabs: ContextTabHost
  readonly vault: ContextVaultHost
  readonly ui: ContextUiHost
  readonly work: ContextWorkHost
}

/** What the host runs with: the deps of the window, with the one reader of its conditions. */
export interface HostDeps extends PluginHostDeps {
  readonly commands: HostCommands & { readonly whenReader: WhenReader }
}
