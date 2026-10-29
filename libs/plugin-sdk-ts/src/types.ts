import type { AgentHost } from './agent'
import type { BufferHost } from './buffers'
import type { DropHost, DropRegistration } from './drops'
import type { NoticeHost } from './notices'
import type { TaskHost } from './tasks'
import type { Selector, TextMaker, VaultTexts } from './texts'
import type { MountedView, ViewRegistration } from './views'
import type { NoteType, VaultFiles, VaultNotes, WorkspaceHost } from './workspace'

// Disposable is a handle released on teardown.
export interface Disposable {
  dispose(): void
}

// TextRange is a run of text, by where it begins and where it ends.
export interface TextRange {
  readonly start: number
  readonly end: number
}

// Result is the discriminated outcome of a fallible operation.
import type { Result } from './result'
export type { Failure, Result } from './result'

// SettingOption declares one choice in an enumerated setting.
export interface SettingOption {
  value: string
  /** The display title for the value; when absent the value is displayed. */
  title?: string
  /** Text displayed under the setting while the value is selected. */
  detail?: string
  /** The ids of the dependencies the host provisions only while the value is in force. */
  dependencies?: string[]
}

// SettingCondition holds while a setting holds one of the values listed.
export interface SettingCondition {
  setting: string
  is: string[]
}

// SettingDeclaration declares a plugin configuration option.
export interface SettingDeclaration {
  key: string
  type: 'string' | 'boolean' | 'number' | 'range' | 'enum' | 'select' | 'agent' | 'file' | 'folder'
  title: string
  description?: string
  default?: unknown
  options?: (string | SettingOption)[]
  /** The options under each value of another enum setting; the first of a list is its default. */
  optionsBy?: { setting: string; options: Record<string, (string | SettingOption)[]> }
  /** The condition under which the setting is shown and handed to the plugin. */
  when?: SettingCondition
  /** The extensions a setting of type file takes, each with its leading dot. */
  accept?: string[]
  min?: number
  max?: number
  step?: number
}

// ReaderRegistration registers a viewer component for files. The manifest declares, under
// `readers` with the same id, the content types it opens. The component gets a ReaderTabState as
// its `state` prop.
export interface ReaderRegistration {
  id: string
  component: unknown
  title?: string
  // The icon its tabs carry before their title, and the files it opens carry in the tree.
  icon?: unknown
  capabilities?: readonly string[]
  metadata?: Record<string, unknown>
}

// SettingsViewRegistration registers a component the plugin's settings page mounts. The manifest
// declares, under `settingsViews` with the same id, its title and the setting it follows.
export interface SettingsViewRegistration {
  readonly id: string
  readonly component: unknown
}

// SettingsViewProps are the props the settings page passes the component of a settings view. The
// component emits `update:setting` with a key and a value; the host writes it to that setting.
export interface SettingsViewProps {
  readonly pluginId: string
  /** The current value of every declared setting of the plugin, by key. */
  readonly values: Readonly<Record<string, unknown>>
}

// ReaderTabState is the `state` prop the window passes the component of a reader tab.
export interface ReaderTabState {
  readonly path: string
  readonly title: string
  // The location to show: the selector of a search hit, or the location saved before a restart. The
  // window sets a new object each time it requests a location; null shows the file from its start.
  readonly selector: Selector | null
  // Saves the place the reader shows. The window keeps it with the tab and passes it back as
  // `selector` when the tab opens after a restart.
  savePlace(place: Selector): void
}

// ReaderPaneProps describes the props passed to a reader component by the host pane wrapper.
export interface ReaderPaneProps {
  readonly path: string
  readonly file?: string
  readonly pluginId?: string
  readonly title?: string
  readonly state?: unknown
}

// TabTarget identifies the tab a command applies to.
export interface TabTarget {
  readonly path?: string
  readonly title?: string
  readonly file?: string
}

// TabPlacement is where a tab of a kind opens in a window that starts with nothing to restore.
export interface TabPlacement {
  // The main pane, or the pane along the trailing edge.
  readonly pane: 'main' | 'aside'
  // The tab's place among the tabs of that pane, lowest first. The first one is in front.
  readonly order: number
}

// TabKindRegistration registers a kind of workspace tab. A registration that
// declares open owns the content of its tabs; one that does not displays the
// note at the path its tab opens on. A kind that declares getState has its
// tabs reopened after a restart with the saved state.
export interface TabKindRegistration<State = unknown> {
  kind: string
  component: unknown
  title?: string
  actions?: Record<string, (targetPath: string, ...args: unknown[]) => unknown>
  onClose?(state: State, id: string): boolean
  // What the tab `id` holds, made as it opens on `at`, inside the tab's effect scope. `saved` is
  // what getState returned for the tab before a restart, and absent for a new tab.
  open?(targetPath: string, id: string, saved?: unknown): State
  // The JSON value the window keeps for the tab and passes to open after a restart.
  getState?(state: State): unknown
  getTitle?(state: State): string
  // The identity a tab takes from what it opens on: the same `at` opened twice is one tab.
  identity?(targetPath: string): string
  onShow?(state: State, id: string): void
  // The tab is handed text meant for it, such as a question the window puts to it.
  receiveText?(state: State, text: string): void
  // A key struck while the tab is in front, answered with whether the tab took it.
  onKeyPress?(state: State, event: KeyboardEvent): boolean
  getTarget?(state: State): TabTarget
  getOpenTab?(state: State): { readonly path: string }
  // The window is going; what the tab holds is released without asking.
  onDestroy?(state: State, id: string): void
  // The tab moves to a vault path a tab linked to it moved to. A kind that declares it can be linked.
  navigate?(state: State, path: string, title: string): void | Promise<void>
  // Its tabs stand beside files, and a file opened on a path never closes one of them.
  isTool?: boolean
  // The one word its tabs carry beside their title, and nothing where a tab carries none.
  getMark?(state: State): string | undefined
  isAsking?(state: State): boolean
  // Writes what the tab holds out to its file, and answers once nothing is on its way.
  flushPending?(state: State): Promise<void>
  // A file made in a folder under a name, as the file this kind edits.
  createFile?(folder: string, name: string): Promise<Result<{ path: string }, string>>
  // The icon its tabs carry before their title.
  icon?: unknown
  placement?: TabPlacement
}

// CommandNote is the note a command runs over: the note in front, or the one a menu was opened on.
// Its path is empty where there is none.
export interface CommandNote {
  readonly path: string
  readonly title: string
  readonly type: NoteType
}

// NOTE_SCOPE offers a command while a note is in front, in the palette's group of note commands.
export const NOTE_SCOPE = 'note' as const

// NOTE_TARGET offers a context menu item on a note.
export const NOTE_TARGET = 'note' as const

// CommandRegistration registers a palette or keybound action.
export interface CommandRegistration {
  id: string
  title: string
  handler: (note: CommandNote) => void | Promise<void>
  shortcut?: string
  canShowInPalette?: boolean
  // 'note' offers the command only while a note is in front, in the palette's group of note commands.
  scope?: 'note'
  // The icon the command is drawn with wherever it is offered.
  icon?: unknown
  // With scope 'note': the command opens a note in the editor. The window runs it for a note
  // selected in the search, navigated to, or named by the agent, and opens the note's file when no command declares one.
  isNoteDestination?: boolean
}

// AgentProviderRegistration registers an AI agent provider.
export interface AgentProviderRegistration {
  id: string
  name: string
  createAgent: (options?: unknown) => unknown
  capabilities?: string[]
}

// PluginLifecycle declares the activation and teardown hooks.
export interface PluginLifecycle {
  activate(context: PluginContext): void | Promise<void>
  deactivate?(): void | Promise<void>
}

// VaultChangeEvent describes a change in vault root or files.
export interface VaultChangeEvent {
  readonly vaultPath?: string
  readonly paths?: readonly string[]
  readonly renamed?: readonly { readonly from: string; readonly to: string }[]
}

// FileSelectEvent describes a file selection or open request.
export interface FileSelectEvent {
  readonly path: string
  readonly title?: string
}

// ContextMenuTarget describes the subject the context menu is opened on.
export interface ContextMenuTarget {
  readonly path: string
  readonly name?: string
  readonly isFolder: boolean
  readonly isNote: boolean
  readonly isSeveral?: boolean
}

// ContextMenuRegistration registers an item to be contributed to the context menu.
export interface ContextMenuRegistration {
  readonly id: string
  readonly text: string
  readonly detail?: string
  readonly group?: string
  /** Submenu parent identifier (e.g. 'new' or 'relation'). */
  readonly parentId?: string
  /** Which target kinds this item applies to. */
  readonly target?: 'all' | 'file' | 'folder' | 'root' | 'note' | readonly string[]
  readonly order?: number
  readonly isDisabled?: boolean // The icon the item is drawn with.
  readonly icon?: unknown | ((target: ContextMenuTarget) => boolean)
  readonly handler: (target: ContextMenuTarget) => void | Promise<void>
}

// PluginContext provides registration and host services to an activated plugin.
export interface PluginContext {
  readonly pluginId: string
  // The window the plugin is activated in, named as a manifest's targets name it.
  readonly target: string
  readonly subscriptions: Disposable[]
  registerReader(registration: ReaderRegistration): Disposable
  // Registers the component of the settings view the manifest declares under the same id.
  registerSettingsView(registration: SettingsViewRegistration): Disposable
  registerTabKind<State>(registration: TabKindRegistration<State>): Disposable
  registerCommand(registration: CommandRegistration): Disposable
  registerContextMenu(registration: ContextMenuRegistration): Disposable
  registerAgentProvider(registration: AgentProviderRegistration): Disposable
  // Declares a view other plugins mount by its capability.
  registerView(registration: ViewRegistration): Disposable
  // Draws the view a plugin declares for `capability` into `element`, passed `context`. While no
  // plugin declares one the element stays empty, and the view is drawn once one does.
  mountView(capability: string, element: HTMLElement, context: unknown): MountedView
  getSetting<T = unknown>(key: string): T | undefined
  onSettingChange<T = unknown>(key: string, handler: (value: T) => void): Disposable
  onVaultChange(handler: (event: VaultChangeEvent) => void | Promise<void>): Disposable
  onFileSelect(handler: (event: FileSelectEvent) => void | Promise<void>): Disposable
  emitVaultChange(event: VaultChangeEvent): void
  emitFileSelect(event: FileSelectEvent): void
  openTab?(kind: string, at?: string): Promise<string> | void
  // Brings the tab of the kind last used to the front and navigates it to `path`; opens a tab of that kind at `path` when none is open.
  navigateTab?(kind: string, path: string): Promise<string> | void
  closeTab?(id: string): void
  hasPermission?(permission: string): boolean
  // The window the plugin's tabs are drawn in, where the host offers one.
  readonly workspace?: WorkspaceHost
  // The notes of the vault, where the host offers them.
  readonly notes?: VaultNotes
  // The folders and files of the vault, where the host offers them.
  readonly files?: VaultFiles
  // The agent the window talks to, when the host provides one.
  readonly agent?: AgentHost
  // The open files of the window, where the host offers them.
  readonly buffers?: BufferHost
  // The text versions of the vault the window shows, where the host offers them.
  readonly texts?: VaultTexts
  // The notices of the window, where the host offers them.
  readonly notices?: NoticeHost
  // The tasks the plugins report, where the host offers them.
  readonly tasks?: TaskHost
  // Declares the data dragged in from outside the window the plugin takes, where the host offers drops.
  registerDropHandler?(registration: DropRegistration): Disposable
  // Hands a drop to the plugin that takes it, where the host offers drops.
  readonly drops?: DropHost
}

// PluginManifest describes the metadata in manifest.json.
export interface PluginManifest {
  id: string
  name: string
  version: string
  /** The plugin protocol the plugin was written against; absent means 1. */
  protocol?: number
  description?: string
  capabilities?: string[]
  targets?: string[]
  activationEvents?: string[]
  permissions?: string[]
  runtimes?: string[]
  /** The secrets the plugin asks the host to store and pass to it in Configure. */
  secrets?: { name: string; title: string; description?: string; when?: SettingCondition }[]
  lifecycle?: {
    idleTimeoutSeconds?: number
    keepAliveWhileVisible?: boolean
  }
  backend?: {
    type: 'process'
    entrypoint: string
    serviceName?: string
  }
  frontend?: {
    entrypoint: string
    styles?: string
  }
  /** What the plugin declares of the text it makes; a plugin that makes no text declares none. */
  /** The types of file the plugin brings, each with the globs of its names, such as `*.deck`. */
  contentTypes?: { type: string; globs: string[] }[]
  /** The readers of the frontend, by registered id, with the content types each opens and its rank. */
  readers?: { id: string; contentTypes: string[]; rank: 'default' | 'alternate' }[]
  texts?: TextMaker
  settings?: SettingDeclaration[]
  /** The views the settings page mounts in the plugin's form, after the setting named by `after`; the top where it is empty. */
  settingsViews?: { id: string; title: string; after?: string }[]
}
