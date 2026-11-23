/**
 * Types and interfaces for the plugin subsystem.
 */
import type {
  AppInfo,
  Disposable,
  BackgroundTaskHost,
  BufferHost,
  CommandAnswers,
  CommandItem,
  CommandWarns,
  CommandsHost,
  FilePlacement,
  JsonValue,
  LogHost,
  NoticeHost,
  PickerHost,
  ProcessHost,
  EditorSettings,
  CommandTarget,
  TaskHost,
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
import type { DeclaredSlotEntry, DeclaredSlotPoint, SlotCommandLook, SlotLayout } from '../slots'
import type { WhenValues, WhenReader } from '../when'
import type { HealthReports } from '../pluginHealth'
import type { Component } from 'vue'

/** One command as a plugin manifest declares it under `contributes.commands`, with the parts its plugin has. */
export interface CommandDeclaration {
  readonly pluginId: string
  readonly id: string
  readonly title: string
  readonly icon?: string
  readonly when?: string
  readonly permission?: string
  readonly answers?: CommandAnswers
  readonly warns?: CommandWarns
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
interface BackendHandlerRun {
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
  readonly answers?: CommandAnswers
  readonly warns?: CommandWarns
}

/** What a slot's command finder reads of the commands the plugins registered with the host. */
interface HostCommandReads {
  readonly getPluginCommands: () => readonly PluginPaletteCommand[]
  readonly hasPluginItems: (command: string) => boolean
}

type SlotCommandFinder = (
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
 * commands, frame, notices and picker.
 */
export interface HostWindow {
  readonly target: string
  readonly app: AppInfo
  readonly openExternal: (url: string) => Promise<void>
  readonly onError: (pluginId: string, error: unknown) => void
  readonly getWorkspace: () => WorkspaceHost | undefined
  readonly getCommands: () => CommandsHost | undefined
  readonly getWindow: () => WindowHost | undefined
  readonly getNotices: (pluginId: string) => NoticeHost
  readonly getPicker: () => PickerHost | undefined
  readonly clearNotices: (pluginId: string) => void
}

/**
 * The plugins of the window: the parts each has, whether its start failed, the starts and the
 * settings. startPlugin settles once the plugin is active or has failed, and does nothing for a
 * plugin that is off; findPluginsOf matches the manifests read so far.
 */
export interface HostPlugins {
  readonly hasPluginFrontend: (pluginId: string) => boolean
  readonly hasPluginBackend: (pluginId: string) => boolean
  readonly isPluginFailed: (pluginId: string) => boolean
  readonly startPlugin: (pluginId: string) => Promise<void>
  readonly startPluginsOf: (match: ContributionMatch) => Promise<void>
  readonly findPluginsOf: (match: ContributionMatch) => readonly string[]
  readonly getSetting: (pluginId: string, key: string) => JsonValue | undefined
  readonly watchSetting: (
    pluginId: string,
    key: string,
    handler: (value: JsonValue | undefined) => void | Promise<void>,
  ) => Disposable
  readonly getEditorSettings: () => EditorSettings | undefined
  readonly watchEditorSettings: (
    pluginId: string,
    handler: (settings: EditorSettings) => void | Promise<void>,
  ) => Disposable
  readonly hasLicense: (pluginId: string, licenseId: string) => boolean
  readonly getLicenseKey: (pluginId: string, licenseId: string) => string | undefined
  readonly watchLicense: (
    pluginId: string,
    licenseId: string,
    handler: (isActive: boolean) => void | Promise<void>,
  ) => Disposable
}

/** The commands the plugins declare, the handlers their backends run, and what their conditions read. */
export interface HostCommands {
  readonly getCommandDeclarations: () => readonly CommandDeclaration[] | undefined
  readonly getHandlerDeclarations: () => readonly HandlerDeclaration[] | undefined
  readonly listBackendHandlerItems: (
    pluginId: string,
    command: string,
    target: CommandTarget,
  ) => Promise<readonly CommandItem[]>
  readonly runBackendHandler: (run: BackendHandlerRun) => Promise<void>
  readonly reportCommandIssue: (pluginId: string, message: string) => void
  readonly getContentTypes: () => Readonly<Record<string, string>>
  readonly getWindowValues: () => WhenValues | undefined
  readonly whenReader?: WhenReader
}

/**
 * The slots of the window and the views drawn in them. getSlotEntries reads the slot entries from the
 * manifests of the plugins whose context the host made, keyed by plugin id.
 */
export interface HostSlots {
  readonly getSlotEntries: (
    manifests: ReadonlyMap<string, Manifest>,
  ) => readonly DeclaredSlotEntry[] | undefined
  readonly getSlotPoints: (
    manifests: ReadonlyMap<string, Manifest>,
  ) => readonly DeclaredSlotPoint[] | undefined
  readonly getSlotLayout: (pluginId: string) => SlotLayout | undefined
  readonly createSlotCommandFinder: (own: HostCommandReads) => SlotCommandFinder
  readonly getSlotWhenValues: (target: CommandTarget | undefined) => WhenValues | undefined
  readonly reportSlotIssue: (
    pluginId: string,
    message: string,
    severity: 'warning' | 'error',
  ) => void
  readonly reportViewIssue: (pluginId: string, message: string) => void
}

/**
 * The tab kinds and readers the plugins bring into the window, the tabs of a plugin that stops, and
 * the notes the tab kinds of the plugins show.
 */
export interface HostKinds {
  readonly registerKind: (kind: AnyTabKind) => () => void
  readonly registerExtensionReader: (
    extension: string,
    readItem: SourceReader,
    icon?: Component,
  ) => () => void
  readonly getReaderExtensions: (pluginId: string, readerId: string) => readonly string[]
  readonly registerEditor: (
    type: string,
    open: (path: string, title?: string, options?: FilePlacement | EditorPlacement | null) => void,
  ) => (() => void) | void
  readonly closeTabsForPlugin: (pluginId: string) => void
  readonly getNoteBuffer: () => NoteBuffer | undefined
  readonly readNote:
    ((path: string) => Promise<{ body?: string; error?: string } | string | undefined>) | undefined
  readonly writeNote: ((path: string, body: string) => Promise<void>) | undefined
}

/** The tabs of the window. */
export interface HostTabs {
  readonly openTab: (kind: string, at?: string) => Promise<string> | undefined
  readonly openTabAdjacent: (kind: string, at?: string) => Promise<string> | undefined
  readonly navigateTab: (kind: string, path: string) => Promise<string> | undefined
  readonly closeTab: (id: string) => void
  readonly getTabState: (id: string) => object | undefined
}

/** The vault services of the window, and how it resolves a link between notes. */
export interface HostVault {
  readonly getNotes: () => VaultNotes | undefined
  readonly getRelations: () => VaultRelations | undefined
  readonly getFiles: () => VaultFiles | undefined
  readonly getVaults: () => VaultsHost | undefined
  readonly getBuffers: () => BufferHost | undefined
  readonly getTexts: (pluginId: string) => VaultTexts | undefined
  readonly resolveLink: (from: string, address: string) => Promise<string>
}

/** The background work of the window: the tasks, the connections, the processes, the logs and the health items. */
export interface HostWork {
  readonly getTasks: (pluginId: string) => TaskHost | undefined
  readonly getConnections: (pluginId: string) => ConnectionHost | undefined
  readonly getBackgroundTasks: () => BackgroundTaskHost | undefined
  readonly getProcesses: () => ProcessHost | undefined
  readonly getLogs: (pluginId: string) => LogHost | undefined
  readonly health: HealthReports
}

/** What the plugin host is made from, by the responsibility of each part. */
export interface PluginHostDeps {
  readonly window: HostWindow
  readonly plugins: HostPlugins
  readonly commands: HostCommands
  readonly slots: HostSlots
  readonly kinds: HostKinds
  readonly tabs: HostTabs
  readonly vault: HostVault
  readonly work: HostWork
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
    readonly isReadOnly?: boolean
  }
  getErrorMessage(id: string): string
}

/** What the host runs with: the deps of the window, with the one reader of its conditions. */
export interface HostDeps extends PluginHostDeps {
  readonly commands: HostCommands & { readonly whenReader: WhenReader }
}

export type {
  ContextEventHost,
  ContextRegistrationHost,
  ContextSettingHost,
  ContextUiHost,
  ContextViewHost,
  ContextWorkHost,
  PluginContextHost,
} from './contextHostTypes'
