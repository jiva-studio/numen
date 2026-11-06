import type { AgentHost } from './agent'
import type { BufferHost } from './buffers'
import type { DropHost, DropRegistration } from './drops'
import type { NoticeHost } from './notices'
import type { TaskHost } from './tasks'
import type { BackgroundTaskHost } from './backgroundTasks'
import type { SlotHost } from './slots'
import type { VaultsHost, WindowHost } from './window'
import type { Selector, VaultTexts } from './texts'
import type { MountedView, ViewRegistration } from './views'
import type { NoteType, VaultFiles, VaultNotes, VaultRelations, WorkspaceHost } from './workspace'

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
export type { Result } from './result'

// ReaderRegistration registers a viewer component for files. The manifest declares, under
// `contributes.readers` with the same id, the content types it opens. The component gets a ReaderTabState as
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
// declares, under `settingsViews` with the same id, its title.
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

// StatusViewRegistration registers a component the window draws in its status bar. The manifest
// declares the same id under `contributes.statusViews`. The component gets StatusViewProps as props.
export interface StatusViewRegistration {
  readonly id: string
  readonly component: unknown
}

// StatusViewProps are the props the window passes the component of a status view.
export interface StatusViewProps {
  readonly pluginId: string
}

// SlotViewRegistration registers a component the window draws in one of its slots. The manifest
// declares, under `contributes.views` with the same id, the slot it is drawn in.
export interface SlotViewRegistration {
  readonly id: string
  readonly component: unknown
}

// TITLE_BAR_SLOT is the slot at the top of a window. Its view draws the whole bar: the commands, the
// title, the window buttons and the area that drags the window. The component gets TitleBarViewProps
// as props and emits TitleBarViewEvents.
export const TITLE_BAR_SLOT = 'titleBar'

// TitleBarCommand is one command placed in `titleBar.commands` of `contributes.menus`, as the window passes it
// to the title bar view.
export interface TitleBarCommand {
  // The full id of the command; the view emits it with `run-command`.
  readonly id: string
  readonly title: string
  // The icon component of the command.
  readonly icon: unknown
  // Commands that share a group are drawn together.
  readonly group?: string
  // The shortcut as the person reads it, such as `Ctrl+Shift+E`.
  readonly shortcut?: string
}

// TitleBarVault is one entry of the list of vaults the title opens.
export interface TitleBarVault {
  // The id the view emits with `switch-vault`.
  readonly id: string
  readonly name: string
  // Whether the entry is the vault the window shows.
  readonly isCurrent?: boolean
}

// TitleBarViewProps are the props the window passes the component of the title bar view.
export interface TitleBarViewProps {
  readonly pluginId: string
  readonly appName: string
  // The vault the window shows; empty where none is open.
  readonly vaultName: string
  // The operating system the window runs on.
  readonly platform: AppInfo['platform']
  readonly isMaximized: boolean
  // True while the window fills the screen; macOS then shows no system buttons.
  readonly isFullscreen: boolean
  // In the order the menus sort them.
  readonly commands: readonly TitleBarCommand[]
  // The vaults the title offers to switch to; empty while no vault is open.
  readonly vaults: readonly TitleBarVault[]
}

// TitleBarViewEvents are the events the window answers from the component of the title bar view.
export interface TitleBarViewEvents {
  // Runs the command of the id.
  'run-command': [id: string]
  // Shows the vault of the id in the window.
  'switch-vault': [id: string]
  'minimize-window': []
  'toggle-maximize-window': []
  'close-window': []
}

// AppInfo describes the application build a plugin runs in.
export interface AppInfo {
  // The name of the application, as a window title names it.
  readonly name: string
  // The version the build calls itself, such as `2026.10.3` or `0.0.0-dev` for a manual build.
  readonly version: string
  // The build number; 0 where the build has none.
  readonly build: number
  readonly platform: 'darwin' | 'windows' | 'linux'
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

// TabKindRegistration registers a workspace tab kind. A registration that
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

// CommandRegistration binds the handler of a command the manifest declares under `contributes.commands`.
// The title, icon, scope and shortcut are read from the declaration. A handler registered for an id the
// manifest does not declare is never listed.
export interface CommandRegistration {
  // The id as `<object>.<action>`, such as `tree.open`, as the manifest declares it. The host files
  // the command under `<plugin id>:<id>`; registering the same full id twice throws.
  id: string
  handler: (note: CommandNote) => void | Promise<void>
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

// PluginContext provides registration and host services to an activated plugin.
export interface PluginContext {
  readonly pluginId: string
  // The window the plugin is activated in, named as a manifest's targets name it.
  readonly target: string
  readonly subscriptions: Disposable[]
  // The application build the plugin runs in.
  readonly app: AppInfo
  // Opens an http or https address in the system browser. Any other address fails.
  openExternal(url: string): Promise<Result<null>>
  registerReader(registration: ReaderRegistration): Disposable
  // Registers the component of the settings view the manifest declares under the same id.
  registerSettingsView(registration: SettingsViewRegistration): Disposable
  // Registers the component of the status bar view the manifest declares under the same id.
  registerStatusView(registration: StatusViewRegistration): Disposable
  // Registers the component of the view the manifest declares under the same id in
  // `contributes.views`, drawn in the slot the manifest names. An id the manifest does not declare
  // throws.
  registerSlotView(registration: SlotViewRegistration): Disposable
  registerTabKind<State>(registration: TabKindRegistration<State>): Disposable
  registerCommand(registration: CommandRegistration): Disposable
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
  // Brings the most recently active tab of the kind to the front and navigates it to `path`; opens a tab of that kind at `path` when none is open.
  navigateTab?(kind: string, path: string): Promise<string> | void
  closeTab?(id: string): void
  hasPermission?(permission: string): boolean
  // The window the plugin's tabs are drawn in, where the host offers one.
  readonly workspace?: WorkspaceHost
  // The notes of the vault, where the host offers them.
  readonly notes?: VaultNotes
  // The relationships between the notes of the vault, where the host offers them.
  readonly relations?: VaultRelations
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
  // The slots of the window: the entries of a slot, read by its owner, and their commands.
  readonly slots?: SlotHost
  // The vaults the window can show, where the host offers them.
  readonly vaults?: VaultsHost
  // The window the plugin runs in, where the host offers it.
  readonly window?: WindowHost
  // The background work of the window, where the host offers it.
  readonly backgroundTasks?: BackgroundTaskHost
  // Declares the data dragged in from outside the window the plugin takes, where the host offers drops.
  registerDropHandler?(registration: DropRegistration): Disposable
  // Hands a drop to the plugin that takes it, where the host offers drops.
  readonly drops?: DropHost
}
