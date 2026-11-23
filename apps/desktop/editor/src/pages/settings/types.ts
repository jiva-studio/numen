/**
 * The settings tab, and its contents.
 *
 * It holds nothing of its own: the window already keeps every setting it can
 * write, and this is a second way to the same values. A row here and the
 * command of the same name in the palette go through one piece of code.
 */
import type { Ref } from 'vue'
import type { CatalogEntry, PluginHealthItem } from '@/entities/plugin'
import type {
  ConnectionChange,
  ConnectionDraft,
  ConnectionInfo,
  ConnectionTypeInfo,
  OperationTitles,
  RowOption,
} from '@/entities/connection'
import type { IdentityInfo, KitProof, RecoveryKit } from '@/entities/identity'
import type { DeviceCalls } from '@/features/identity-link'
import type { Printer } from '@/pages/settings/api/print'
import type { LicenseInfo } from '@/entities/license'
import type {
  PluginInfo,
  PluginSettingGroupInfo,
  PluginSettingInfo,
  SettingEdit,
} from '@/entities/settings'
import type { JsonValue } from '@/shared/api/json'
import type { Result } from '@numen/result'
import type { Component } from 'vue'

/** What this installation is configured as, as the window already holds it. */
export interface Installation {
  readonly settings: InstallationSettings
  readonly plugin: InstallationPlugin
  readonly connections: InstallationConnections
  readonly identity: InstallationIdentity
  readonly printer: Printer
  readonly licenses: InstallationLicenses
}

/**
 * The settings of the installation, read and written in place. `getPlatform` answers the operating
 * system of the window: darwin, windows or linux; the `app.platform` of conditions.
 */
/** The component a plugin registered for a view of its settings page. */
export interface ViewComponent {
  readonly component: Component
}

/** A view a plugin registered for the page "Identity and licenses", and the plugin it belongs to. */
export interface IdentityView extends ViewComponent {
  readonly pluginId: string
  readonly id: string
}

export interface InstallationSettings {
  getSetting(at: readonly string[]): JsonValue | undefined
  /** A setting as the vault last answered it, never a value asked for and not yet answered. */
  getConfirmedSetting(at: readonly string[]): JsonValue | undefined
  /** Whether the write of a setting is in flight. */
  isPending(at: readonly string[]): boolean
  /**
   * Whether the value requested at a path is not yet in the declaration the window holds: from the
   * request until the read that carries it is taken.
   */
  isRequested(at: readonly string[]): boolean
  getPlatform(): string
  write(content: readonly SettingEdit[]): void
  readonly file: Readonly<Ref<string>>
}

/** One entry a slot holds, named by the key a stored layout refers to it by. */
export interface SlotContributor {
  readonly group?: string | undefined
  readonly key: string
  readonly label: string
  readonly pluginName: string
}

/** A slot a plugin draws and its entries, in the order the slot draws them when nothing is stored. */
export interface SlotGroup {
  readonly slot: string
  readonly title: string
  readonly contributors: readonly SlotContributor[]
}

/**
 * The plugins of the installation, what they declare and what the window does for them.
 * `getSettingsViews` answers the components a plugin's frontend registered for its settings
 * views, by view id; a view the plugin declares and has not registered is absent.
 */
export interface InstallationPlugin {
  readonly plugins: Readonly<Ref<readonly PluginInfo[]>>
  readonly app: Readonly<Ref<readonly PluginSettingInfo[]>>
  readonly appGroups: Readonly<Ref<readonly PluginSettingGroupInfo[]>>
  readonly catalog: Readonly<Ref<readonly CatalogEntry[]>>
  readonly health: InstallationHealth
  /** A plugin whose last start failed, started once more. */
  retryPlugin(pluginId: string): void
  /**
   * Opens the machine's native picker for a plugin's setting of type file or folder, and
   * writes the chosen path. Resolves to an empty string where the path was written or none
   * was chosen.
   */
  choosePath(pluginId: string, key: string): Promise<string>
  getSettingsViews(pluginId: string): ReadonlyMap<string, ViewComponent>
  /** The views of the enabled plugins that belong to the page "Identity and licenses", in the order of the plugins. */
  getIdentityViews(): readonly IdentityView[]
  /** The slots a plugin draws, with what is placed in each; absent where the window offers none. */
  getSlotGroups?(pluginId: string): readonly SlotGroup[]
  readonly files: InstallationPluginFiles
}

/** The health items of the plugins, and the commands their fixes run. */
interface InstallationHealth {
  readonly items: Readonly<Ref<readonly PluginHealthItem[]>>
  readonly canRunCommand: (command: string) => boolean
  readonly runCommand: (command: string, values: Readonly<Record<string, string>>) => void
}

/** What a plugin keeps and serves as files. */
interface InstallationPluginFiles {
  /** How many files the folder of a plugin holds in the shown vault, `.numen/<plugin id>/`. */
  getPluginData(pluginId: string): Promise<number>
  /**
   * That folder deleted, with the text versions the plugin made there. The answer is why it was not
   * deleted, and empty where it was.
   */
  deletePluginData(pluginId: string): Promise<string>
  /** The README.md the plugin serves, and an empty text where it has none. */
  getReadme(pluginId: string): Promise<string>
}

/**
 * The connections to services and programs, and what the rows of connections on a plugin page are
 * filled from. A value of a secret is entered once and is never read back.
 */
export interface InstallationConnections {
  readonly list: Readonly<Ref<readonly ConnectionInfo[]>>
  readonly types: Readonly<Ref<readonly ConnectionTypeInfo[]>>
  readonly operationTitles: Readonly<Ref<OperationTitles>>
  /** Reads the types and the connections. */
  load(): Promise<void>
  readonly edits: ConnectionEdits
  /**
   * The values a row field of the type of a connection can take, from the provider of its type, for the
   * values the row holds so far. `operations` are the operations the setting of rows is used for.
   */
  listRowOptions(
    connectionId: string,
    field: string,
    options: Readonly<Record<string, string>>,
    operations: readonly string[],
  ): Promise<Result<readonly RowOption[], string>>
  /** How many notes the shown vault holds. Rows of connections for embeddings say it before they index. */
  countNotes(): Promise<number>
}

/** The changes to the stored connections; each answers the connection as stored, or why it was not. */
export interface ConnectionEdits {
  readonly create: (draft: ConnectionDraft) => Promise<Result<ConnectionInfo, string>>
  readonly update: (change: ConnectionChange) => Promise<Result<ConnectionInfo, string>>
  readonly delete: (id: string) => Promise<Result<undefined, string>>
}

/**
 * The identity of this device. The recovery words and the sheets of a kit pass through the calls that
 * make or take them and are not held here.
 */
export interface InstallationIdentity {
  readonly identity: Readonly<Ref<IdentityInfo | undefined>>
  readonly devices: DeviceCalls
  load(): Promise<void>
  generateWords(): Promise<Result<readonly string[], string>>
  create(words: readonly string[]): Promise<Result<IdentityInfo, string>>
  restore(words: readonly string[]): Promise<Result<IdentityInfo, string>>
  delete(): Promise<Result<void, string>>
  createKit(proof: KitProof): Promise<Result<RecoveryKit, string>>
  checkKit(sheets: readonly (readonly string[])[]): Promise<Result<string, string>>
  restoreFromKit(sheets: readonly (readonly string[])[]): Promise<Result<IdentityInfo, string>>
}

/**
 * The licenses declared by the installed plugins, and their activation state on this machine.
 */
export interface InstallationLicenses {
  readonly list: Readonly<Ref<readonly LicenseInfo[]>>
  load(): Promise<void>
  watch(signal: AbortSignal): Promise<void>
  activate(licenseId: string, key: string): Promise<Result<LicenseInfo, string>>
  deactivate(licenseId: string): Promise<Result<LicenseInfo, string>>
}

/** One of the choices a select offers. */
export interface SelectChoice {
  readonly id: string
  readonly text: string
  readonly detail?: string
  readonly group?: string
  readonly isDisabled?: boolean
}

/** Asks the provider of a connection for the values of a row field. */
export type ListRowOptions = InstallationConnections['listRowOptions']

/** What the fields of one plugin ask of the plugins of the installation. */
export type PluginFieldsHost = Pick<InstallationPlugin, 'choosePath' | 'getSettingsViews'>

/** A setting of one plugin the settings tab was opened at. Each request is a new object. */
export interface MarkedSetting {
  readonly pluginId: string
  readonly setting: string
}

/** What the settings tab holds. */
export interface SettingsTabState {
  readonly installation: Installation
  readonly initialPluginId: Ref<string | null>
  readonly initialSearchQuery: Ref<string>
  readonly markedSetting: Ref<MarkedSetting | null>
}
