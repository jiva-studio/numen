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
  FileSelectEvent,
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
  ViewRegistration,
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
import type { Component } from 'vue'

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
  /** The Lucide name of the icon it is drawn with. */
  readonly icon?: string
  readonly shortcut?: string
  /** The CEL condition under which it is available, as the manifest declares it; absent is always. */
  readonly when?: string
}

/** One handler of a shared command as a plugin manifest declares it under `contributes.handlers`. */
export interface HandlerDeclaration {
  readonly pluginId: string
  /** The full id of the shared command, such as `app:file.createText`. */
  readonly command: string
  readonly title: string
  /** The CEL condition over which targets it handles the command; absent is every target. */
  readonly when?: string
  /** Set where its condition names `version.*`: the core lists the versions it offers as its items. */
  readonly hasItems: boolean
}

export interface PluginPaletteCommand {
  readonly id: string
  readonly title: string
  readonly shortcut?: string
  /** The CEL condition under which it is available; absent is always. */
  readonly when?: string
  /** The Lucide name of the icon it is drawn with. */
  readonly icon?: string
}

export type TabActionHandler = (targetPath: string, ...args: unknown[]) => unknown

export interface PluginHostDeps {
  /** The content type of a file by its extension, as the core lists it, for the conditions of commands. */
  readonly getContentTypes?: () => Readonly<Record<string, string>>
  /** The window and settings variables the conditions of commands read: window.isVaultOpen and the others, and config. */
  readonly getWindowValues?: () => WhenValues
  /** The commands the enabled plugins declare in their manifests; undefined until the first list has loaded. */
  readonly getCommandDeclarations?: () => readonly CommandDeclaration[] | undefined
  /** The handlers of shared commands the enabled plugins declare; undefined until the first list has loaded. */
  readonly getHandlerDeclarations?: () => readonly HandlerDeclaration[] | undefined
  /** The items the core lists for a handler that runs in its plugin's backend, over the target. */
  readonly listBackendHandlerItems?: (
    pluginId: string,
    command: string,
    target: CommandTarget,
  ) => Promise<readonly CommandItem[]>
  /** Runs a handler in its plugin's backend over the target; with `argument`, the id of the item picked. */
  readonly runBackendHandler?: (
    pluginId: string,
    command: string,
    target: CommandTarget,
    argument?: string,
  ) => Promise<void>
  /** Whether the plugin declares a frontend the window loads; where absent, every plugin does. */
  readonly hasPluginFrontend?: (pluginId: string) => boolean
  /** Whether the plugin declares a backend that runs the commands no frontend handler takes; where absent, none does. */
  readonly hasPluginBackend?: (pluginId: string) => boolean
  /** Whether the plugin failed to load or to activate. */
  readonly isPluginFailed?: (pluginId: string) => boolean
  /** Settles once the plugin has been activated, or has failed to. */
  readonly waitForPlugin?: (pluginId: string) => Promise<void>
  /** Reports a command whose declaration and handler do not match. */
  readonly reportCommandIssue?: (pluginId: string, message: string) => void
  /** Reports a view of a slot that another plugin, first by plugin id, holds. */
  readonly reportViewIssue?: (pluginId: string, message: string) => void
  readonly registerKind: (kind: AnyTabKind) => () => void
  readonly registerExtensionReader: (
    extension: string,
    readItem: SourceReader,
    icon?: Component,
  ) => () => void
  /** The extensions of the files the reader of the plugin opens, as the manifest and the core choose. */
  readonly getReaderExtensions?: (pluginId: string, readerId: string) => readonly string[]
  readonly registerEditor?: (
    type: string,
    open: (path: string, title?: string, options?: unknown) => void,
  ) => (() => void) | void
  readonly openTab: (kind: string, at?: string) => Promise<string>
  readonly openTabAdjacent?: (kind: string, at?: string) => Promise<string>
  readonly navigateTab?: (kind: string, path: string) => Promise<string>
  readonly closeTab?: (id: string) => void
  readonly getTabState?: (id: string) => unknown
  readonly getSetting?: (pluginId: string, key: string) => unknown
  readonly onSettingChange?: (
    pluginId: string,
    key: string,
    handler: (value: unknown) => void | Promise<void>,
  ) => () => void
  readonly closeTabsForPlugin?: (pluginId: string) => void
  readonly onError?: (pluginId: string, error: unknown) => void
  readonly tabActionHandlers?: Record<string, TabActionHandler>
  readonly onTabAction?: (action: string, targetPath: string, ...args: unknown[]) => unknown

  readonly readNote?:
    ((path: string) => Promise<{ body?: string; error?: string } | string | undefined>) | undefined
  readonly writeNote?: ((path: string, body: string) => Promise<void>) | undefined
  /** The window a plugin's tabs are drawn in, once the window has made it. */
  readonly getWorkspace?: () => WorkspaceHost | undefined
  /** The notes of the vault, once the window has made them. */
  readonly getNotes?: () => VaultNotes | undefined
  /** The relationships between notes, once the window has made them. */
  readonly getRelations?: () => VaultRelations | undefined
  /** The files of the vault, once the window has made them. */
  readonly getFiles?: () => VaultFiles | undefined
  /** The agent the window queries, once the window has created it. */
  readonly getAgent?: () => AgentHost | undefined
  /** The commands of the window, once the window has made them. */
  readonly getCommands?: () => CommandsHost | undefined
  /** The vaults the window can show, once the window has made them. */
  readonly getVaults?: () => VaultsHost | undefined
  /** The window the plugins run in, once the window has made it. */
  readonly getWindow?: () => WindowHost | undefined
  /** The open files of the window, once the window has made them. */
  readonly getBuffers?: () => BufferHost | undefined
  /** The window's store of open notes, which the tabs of the note kind read and write through. */
  readonly getNoteBuffer?: () => NoteBuffer | undefined
  /** The vault path an address written in a note lands on, empty where it lands on none. */
  readonly resolveLink?: (from: string, address: string) => Promise<string>
  /** The text versions of the shown vault, as the views of one plugin read and edit them. */
  readonly getTexts?: (pluginId: string) => VaultTexts
  /** The notices one plugin shows in the window's corner. */
  readonly getNotices?: (pluginId: string) => NoticeHost
  /** The tasks the plugins report, as one plugin watches them. */
  readonly getTasks?: (pluginId: string) => TaskHost | undefined
  /** The entries the enabled plugins and the window place in slots; undefined until the first list has loaded. */
  readonly getSlotEntries?: () => readonly DeclaredSlotEntry[] | undefined
  /** How a command of a slot entry is drawn over the target, and undefined while it is not available there. */
  readonly findSlotCommand?: (
    command: string,
    target: CommandTarget | undefined,
  ) => SlotCommandLook | undefined
  /** The values the when conditions of slot entries read over the target of a slot: target, window and settings. */
  readonly getSlotWhenValues?: (target: CommandTarget | undefined) => WhenValues
  /** Receives what is wrong with the slot entries of a plugin. */
  readonly reportSlotIssue?: (
    pluginId: string,
    message: string,
    severity: 'warning' | 'error',
  ) => void
  /** The background work of the window, as every plugin reads and acts on it. */
  readonly getBackgroundTasks?: () => BackgroundTaskHost | undefined
  /** Everything a plugin showed, taken away with the plugin. */
  readonly clearNotices?: (pluginId: string) => void
  /** The build the window is, as plugins read it from their context. */
  readonly app?: AppInfo
  /** Opens an address in the system browser. */
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

export interface PluginContextHost {
  registerReader(pluginId: string, registration: ReaderRegistration): Disposable
  registerSettingsView(pluginId: string, registration: SettingsViewRegistration): Disposable
  registerSlotView(pluginId: string, slot: string, registration: SlotViewRegistration): Disposable
  readonly app: AppInfo
  openExternal(url: string): Promise<void>
  registerTabKind(pluginId: string, registration: TabKindRegistration): Disposable
  registerCommand(pluginId: string, registration: CommandRegistration): Disposable
  registerAgentProvider(pluginId: string, registration: AgentProviderRegistration): Disposable
  registerView(pluginId: string, registration: ViewRegistration): Disposable
  mountView(capability: string, element: HTMLElement, context: unknown): MountedView
  registerDropHandler(pluginId: string, registration: DropRegistration): Disposable
  readonly drops: DropHost
  getSetting<T = unknown>(pluginId: string, key: string): T | undefined
  onSettingChange<T = unknown>(
    pluginId: string,
    key: string,
    handler: (value: T) => void | Promise<void>,
  ): Disposable
  onVaultChange(
    pluginId: string,
    handler: (event: VaultChangeEvent) => void | Promise<void>,
  ): Disposable
  onFileSelect(
    pluginId: string,
    handler: (event: FileSelectEvent) => void | Promise<void>,
  ): Disposable
  emitVaultChange(pluginId: string, event: VaultChangeEvent): void
  emitFileSelect(pluginId: string, event: FileSelectEvent): void
  openTab?(kind: string, at?: string): Promise<string>
  navigateTab?(kind: string, path: string): Promise<string> | undefined
  closeTab?(id: string): void
  getWorkspace?(): WorkspaceHost | undefined
  getNotes?(): VaultNotes | undefined
  getRelations?(): VaultRelations | undefined
  getFiles?(): VaultFiles | undefined
  getAgent?(): AgentHost | undefined
  getCommands?(): CommandsHost | undefined
  getVaults?(): VaultsHost | undefined
  getWindow?(): WindowHost | undefined
  getBuffers?(): BufferHost | undefined
  getNotices?(pluginId: string): NoticeHost | undefined
  getTasks?(pluginId: string): TaskHost | undefined
  getBackgroundTasks?(): BackgroundTaskHost | undefined
  getTexts?(pluginId: string): VaultTexts | undefined
  getSlots?(): SlotHost | undefined
}
