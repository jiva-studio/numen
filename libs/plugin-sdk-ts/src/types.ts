import type { AgentHost } from './agent'
import type { BufferHost } from './buffers'
import type { DropHost, DropRegistration } from './drops'
import type { NoticeHost } from './notices'
import type { TaskHost } from './tasks'
import type { BackgroundTaskHost } from './backgroundTasks'
import type { CommandItem, CommandsHost, CommandTarget } from './commands/commands'
import type { SlotHost } from './slots/slots'
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

// StatusViewProps are the props the window passes the component of a view in a slot of a bar.
export interface StatusViewProps {
  readonly pluginId: string
}

// SlotViewRegistration registers a component a slot draws. The manifest places it, as an entry
// `{ "slot", "view" }` of `contributes.slots` with the same id, in the slots it is drawn in.
export interface SlotViewRegistration {
  readonly id: string
  readonly component: unknown
}

// TITLE_BAR_SLOT is the slot at the top of a window. Its view draws the whole bar: the commands, the
// title, the window buttons and the area that drags the window. The window passes it no props; it
// reads its slots, the vaults and the window through `slots`, `vaults` and `window` of its context.
export const TITLE_BAR_SLOT = 'titleBar'

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

// TabPlace is the place a reader tab shows, as its reader reports it. A field the reader does not
// have is left out.
export interface TabPlace {
  // PDF: the page shown, from 1, and the number of pages.
  readonly page?: number
  readonly pageCount?: number
  // Recording: seconds from the start.
  readonly positionSeconds?: number
  // EPUB: the CFI of the place shown, and the table-of-contents title there.
  readonly cfi?: string
  readonly chapter?: string
}

// ReaderTabState is the `state` prop the window passes the component of a reader tab.
export interface ReaderTabState {
  readonly path: string
  readonly title: string
  // The location to show: the selector of a search hit, or the location saved before a restart. The
  // window sets a new object each time it requests a location; null shows the file from its start.
  readonly selector: Selector | null
  // Byte ranges of the text version to highlight; the reader scrolls to the first. The window sets a
  // new array each time it requests them; null when none is requested.
  readonly ranges: readonly TextRange[] | null
  // Saves the place the reader shows. The window keeps it with the tab and passes it back as
  // `selector` when the tab opens after a restart.
  savePlace(place: Selector): void
  // Reports the place the reader shows to the window, which an agent reads with the open tabs.
  reportPlace(place: TabPlace): void
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
  // Returns the file the tab shows, and the place its reader last reported, left out when it reported none.
  getOpenTab?(state: State): { readonly path: string; readonly place?: TabPlace }
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

// CommandNote is what a command runs over, as its handler gets it: the target of the run, the note
// in front where the run names none, and the type the vault files it as. Its path is empty where
// there is none.
export interface CommandNote extends CommandTarget {
  readonly type: NoteType
}

// CommandRegistration binds the handler of a command the manifest declares under `contributes.commands`.
// The title, icon, scope and shortcut are read from the declaration. A handler registered for an id the
// manifest does not declare is never listed.
export interface CommandRegistration {
  // The id as `<object>.<action>`, such as `tree.open`, as the manifest declares it. The host files
  // the command under `<plugin id>:<id>`; registering the same full id twice throws. The full id of a
  // shared command, such as `app:file.createText`, registers the plugin's handler of it, which the
  // manifest declares under `contributes.handlers`.
  id: string
  // Runs the command over the note; with `argument`, the id of the item picked.
  handler: (note: CommandNote, argument?: string) => void | Promise<void>
  // The items the command offers over the target, computed each time a place asks; now or later.
  // A command without it offers none.
  listItems?: (target: CommandTarget) => readonly CommandItem[] | Promise<readonly CommandItem[]>
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
  // Registers the component of the view the manifest places under the same id in
  // `contributes.slots`, drawn in every slot the manifest places it in. An id the manifest places
  // nowhere throws.
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
  // The commands of the window: which ones it runs now, and the one way to run a command.
  readonly commands?: CommandsHost
  // The slots of the window: the entries of a slot, read by its owner.
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
