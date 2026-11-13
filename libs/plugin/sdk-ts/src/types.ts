import type { Component, Ref } from 'vue'
import type { FileConflict } from './buffers'
import type { BufferHost } from './buffers'
import type { DropHost, DropRegistration } from './drops'
import type { NoticeHost } from './notices'
import type { TaskHost } from './tasks'
import type { ProcessHost } from './processes'
import type { BackgroundTaskHost } from './backgroundTasks'
import type { HealthHost } from './health'
import type { CommandItem, CommandsHost, CommandTarget } from './commands'
import type { SlotHost } from './slots'
import type { VaultsHost, WindowHost } from './window'
import type { Selector, VaultTexts } from './texts'
import type { ConnectionHost } from './connections'
import type { MountedView, ViewCapability, ViewContexts, ViewRegistration } from './views'
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
  component: Component
  title?: string
  icon?: Component
  capabilities?: readonly string[]
}

// SettingsViewRegistration registers a component the plugin's settings page mounts. The manifest
// declares, under `settingsViews` with the same id, its title.
export interface SettingsViewRegistration {
  readonly id: string
  readonly component: Component
}

// SlotViewRegistration registers a component a slot draws. The manifest places it, as an entry
// `{ "slot", "view" }` of `contributes.slots` with the same id, in the slots it is drawn in.
export interface SlotViewRegistration {
  readonly id: string
  readonly component: Component
}

// AppInfo describes the application build a plugin runs in.
export interface AppInfo {
  readonly name: string
  readonly version: string
  readonly build: number
  readonly platform: 'darwin' | 'windows' | 'linux'
}

// TabPlace is the place a reader tab shows, as its reader reports it. A field the reader does not
// have is left out.
export interface TabPlace {
  readonly page?: number
  readonly pageCount?: number
  readonly positionSeconds?: number
  readonly cfi?: string
  readonly chapter?: string
}

// ReaderHandle is what a reader component lets its tab call: moving the keyboard into the reader, and
// laying it out again once the tab is visible.
export interface ReaderHandle {
  focus(): void
  measure(): void
}

// ReaderTabState is the `state` prop the window passes the component of a reader tab.
export interface ReaderTabState {
  readonly path: string
  readonly title: string
  readonly selector: Selector | null
  readonly ranges: readonly TextRange[] | null
  // Saves the place the reader shows. The window keeps it with the tab and passes it back as
  // `selector` when the tab opens after a restart.
  savePlace(place: Selector): void
  // Reports the place the reader shows to the window, which an agent reads with the open tabs.
  reportPlace(place: TabPlace): void
  // Hands the window the reader, which it focuses and lays out when the tab is shown.
  attachReader?(reader: ReaderHandle | null): void
}

// TabTarget identifies the tab a command applies to.
export interface TabTarget {
  readonly path?: string
  readonly title?: string
  readonly file?: string
}

// TabPlacement is where a tab of a kind opens in a window that starts with nothing to restore.
export interface TabPlacement {
  readonly pane: 'main' | 'aside'
  readonly order: number
}

// TabKindIdentity names a tab kind and the tab an opened path resolves to.
export interface TabKindIdentity<State extends object = object> {
  kind: string
  // The identity a tab takes from what it opens on: the same `at` opened twice is one tab.
  identity?(this: void, targetPath: string): string
  title?: string
  getTitle?(this: void, state: State): string
}

// TabKindPresentation is how the tabs of a kind look and where they open.
export interface TabKindPresentation<State extends object = object> {
  component: Component
  icon?: Component
  placement?: TabPlacement
  isTool?: boolean
  // The one word its tabs carry beside their title, and nothing where a tab carries none.
  getMark?(this: void, state: State): string | undefined
}

// TabKindLifecycle is what a tab holds from its opening to its end.
export interface TabKindLifecycle<State extends object = object> {
  // What the tab `id` holds, made as it opens on `at`, inside the tab's effect scope. `saved` is
  // what getState returned for the tab before a restart, and absent for a new tab.
  open?(targetPath: string, id: string, saved?: JsonValue): State
  // The JSON value the window keeps for the tab and passes to open after a restart.
  getState?(this: void, state: State): JsonValue
  onShow?(this: void, state: State, id: string): void
  onClose?(this: void, state: State, id: string): boolean
  // The window is going; what the tab holds is released without asking.
  onDestroy?(this: void, state: State, id: string): void
}

// TabKindInput is what a tab receives from the window.
export interface TabKindInput<State extends object = object> {
  // The tab is handed text meant for it, such as a question the window puts to it.
  receiveText?(this: void, state: State, text: string): void
  // A key struck while the tab is in front, answered with whether the tab took it.
  onKeyPress?(this: void, state: State, event: KeyboardEvent): boolean
  // The tab moves to a vault path a tab linked to it moved to. A kind that declares it can be linked.
  navigate?(this: void, state: State, path: string, title: string): void | Promise<void>
}

// TabKindFiles is how a tab relates to the files it shows and edits.
export interface TabKindFiles<State extends object = object> {
  getTarget?(this: void, state: State): TabTarget
  // Returns the file the tab shows, and the place its reader last reported, left out when it reported none.
  getOpenTab?(this: void, state: State): { readonly path: string; readonly place?: TabPlace }
  isAsking?(this: void, state: State): boolean
  // Writes what the tab holds out to its file, and answers once nothing is on its way.
  flushPending?(this: void, state: State): Promise<void>
  // A file made in a folder under a name, as the file this kind edits.
  createFile?(this: void, folder: string, name: string): Promise<Result<{ path: string }, string>>
}

// TabKindRegistration registers a workspace tab kind, as the union of its hook groups. A
// registration that declares open owns the content of its tabs; one that does not displays the
// note at the path its tab opens on. A kind that declares getState has its tabs reopened after a
// restart with the saved state.
export interface TabKindRegistration<State extends object = object>
  extends
    TabKindIdentity<State>,
    TabKindPresentation<State>,
    TabKindLifecycle<State>,
    TabKindInput<State>,
    TabKindFiles<State> {}

// CommandNote is what a command runs over, as its handler gets it: the target of the run, the note
// in front where the run names none, and the type the vault files it as. Its path is empty where
// there is none.
export interface CommandNote extends CommandTarget {
  readonly type: NoteType
}

// CommandRegistration binds the handler of a command the manifest declares under `contributes.commands`.
// The title, icon and scope are read from the declaration. A handler registered for an id the
// manifest does not declare is never listed. The handler gets the id of the item picked, and the values of
// the arguments by their declared names, empty when none were given.
export interface CommandRegistration {
  id: string
  handler: (
    note: CommandNote,
    itemId?: string,
    values?: Readonly<Record<string, string>>,
  ) => void | Promise<void>
  listItems?: (target: CommandTarget) => readonly CommandItem[] | Promise<readonly CommandItem[]>
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

// Subscriptions is where a plugin keeps what is released when it stops. A plugin adds to it; only the host
// takes from it.
export interface Subscriptions {
  push(...disposables: Disposable[]): number
  readonly length: number
}

// PluginIdentity is who the plugin is and where it runs.
export interface PluginIdentity {
  readonly id: string
  readonly target: string
  readonly subscriptions: Subscriptions
  readonly app: AppInfo
  // Opens an http or https address in the system browser. Any other address fails; a plugin without
  // the permission link:open (consent to open external links; it does not restrict requests the
  // plugin makes itself) is refused.
  openExternal(url: string): Promise<Result<null>>
  hasPermission?(permission: string): boolean
}

// PluginRegistrations is what a plugin adds to the window.
export interface PluginRegistrations {
  reader(registration: ReaderRegistration): Disposable
  // Registers the component of the settings view the manifest declares under the same id.
  settingsView(registration: SettingsViewRegistration): Disposable
  // Registers the component of the view the manifest places under the same id in
  // `contributes.slots`, drawn in every slot the manifest places it in. An id the manifest places
  // nowhere throws.
  slotView(registration: SlotViewRegistration): Disposable
  tabKind<State extends object>(registration: TabKindRegistration<State>): Disposable
  command(registration: CommandRegistration): Disposable
  // Declares a view other plugins mount by its capability.
  view(registration: ViewRegistration): Disposable
  // Declares the data dragged in from outside the window the plugin takes, where the host offers drops.
  dropHandler?(registration: DropRegistration): Disposable
}

export interface PluginSettings {
  get<T extends JsonValue = JsonValue>(key: string): T | undefined
  onChange<T extends JsonValue = JsonValue>(key: string, handler: (value: T) => void): Disposable
}

// PluginEvents is what a plugin receives of the vault and says to the other plugins.
export interface PluginEvents {
  onVaultChange(handler: (event: VaultChangeEvent) => void | Promise<void>): Disposable
  onFileSelect(handler: (event: FileSelectEvent) => void | Promise<void>): Disposable
  emitVaultChange(event: VaultChangeEvent): void
  emitFileSelect(event: FileSelectEvent): void
}

// PluginTabs is how a plugin opens, moves and closes tabs.
export interface PluginTabs {
  open?(kind: string, at?: string): Promise<string> | void
  // Brings the most recently active tab of the kind to the front and navigates it to `path`; opens a tab of that kind at `path` when none is open.
  navigate?(kind: string, path: string): Promise<string> | void
  close(id: string): void
}

// PluginVault is what a plugin reads and writes of the vault.
export interface PluginVault {
  readonly notes?: VaultNotes
  readonly relations?: VaultRelations
  readonly files?: VaultFiles
  readonly texts?: VaultTexts
  readonly buffers?: BufferHost
  readonly vaults?: VaultsHost
}

// PluginUi is the window a plugin draws in.
export interface PluginUi {
  readonly workspace?: WorkspaceHost
  readonly window?: WindowHost
  readonly commands?: CommandsHost
  readonly slots: SlotHost
  readonly notices: NoticeHost
  readonly drops?: DropHost
  // Draws the view a plugin declares for `capability` into `element`, passed `context`. While no
  // plugin declares one the element stays empty, and the view is drawn once one does.
  mountView<C extends ViewCapability>(
    capability: C,
    element: HTMLElement,
    context: ViewContexts[C],
  ): MountedView
}

// PluginWork is the connections and the background work of the core, and the plugins that run.
export interface PluginWork {
  readonly connections?: ConnectionHost
  readonly tasks?: TaskHost
  readonly backgroundTasks?: BackgroundTaskHost
  readonly processes?: ProcessHost
}

// PluginContext provides registration and host services to an activated plugin, in parts.
export interface PluginContext {
  readonly plugin: PluginIdentity
  readonly register: PluginRegistrations
  readonly settings: PluginSettings
  readonly events: PluginEvents
  readonly tabs: PluginTabs
  readonly vault: PluginVault
  readonly ui: PluginUi
  readonly work: PluginWork
  readonly health?: HealthHost
}

// JsonValue is a value that survives JSON.stringify and JSON.parse unchanged: the shape of a setting,
// of the state a tab keeps across a restart, and of a message to a backend.
export type JsonValue =
  string | number | boolean | null | readonly JsonValue[] | { readonly [key: string]: JsonValue }

export type JsonObject = { readonly [key: string]: JsonValue }

// WikilinkChoice is a note a wikilink can name.
export interface WikilinkChoice {
  readonly title: string
  readonly type: 'note'
}

// EditorHandle is what a note editor lets its tab call: moving the keyboard into it, laying it out
// again, and scrolling to a line. Each answers false where the editor cannot.
export interface EditorHandle {
  focus?(): boolean
  measure?(): void
  reveal?(line: number): boolean
}

// NoteTabState is the `state` prop the window passes the component of a note tab. The refs
// are the host's store of the note: the component reads them and calls the methods to write.
export interface NoteTabState {
  readonly path: string
  readonly body: Readonly<Ref<string>>
  readonly isLoading: Readonly<Ref<boolean>>
  readonly errorMessage: Readonly<Ref<string>>
  readonly fileConflict?: Readonly<Ref<FileConflict>>
  readonly wikilinks?: {
    search(query: string): Promise<readonly WikilinkChoice[]>
    headings(noteTitle: string): Promise<readonly string[]>
  }
  updateBody(text: string): void
  save(): void
  // Keeps the text on screen over the file that changed on disk.
  keepMine?(): void
  // Reads the file that changed on disk, dropping the text on screen.
  takeFile?(): void
  followLink?(address: string): void
  // Hands the window the editor, which it lays out again when the tab is shown.
  attachEditor?(editor: EditorHandle | null): void
  // Lays the attached editor out again; the window calls it where the size of the text changed.
  measure?(): void
}
