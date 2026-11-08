/**
 * The settings tab, and what it holds.
 *
 * It holds nothing of its own: the window already keeps every setting it can
 * write, and this is a second way to the same values. A row here and the
 * command of the same name in the palette go through one piece of code.
 */
import type { Ref } from 'vue'
import type { CatalogEntry } from '@numen/core-client'
import type {
  Model,
  PluginInfo,
  PluginSettingInfo,
  SecretMenu,
  SecretSection,
  SettingEdit,
} from '@/entities/settings'

/** What this installation is configured as, as the window already holds it. */
export interface Installation {
  /**
   * names one can be set to, and settings written where they stand.
   */
  getSetting(at: readonly string[]): unknown
  /** A setting as the vault last answered it, never a value asked for and not yet answered. */
  getConfirmedSetting?(at: readonly string[]): unknown
  /** Whether the write of a setting is in flight. */
  isPending?(at: readonly string[]): boolean
  /**
   * Whether the value requested at a path is not yet in the declaration the window holds: from the
   * request until the read that carries it is taken.
   */
  isRequested?(at: readonly string[]): boolean
  getModels(at: readonly string[]): readonly Model[]
  /** The operating system of the window: darwin, windows or linux; the `app.platform` of conditions. */
  getPlatform?(): string
  write(content: readonly SettingEdit[]): void
  /**
   * The secrets, each a name in a group with the plugin declarations that may read it. A value
   * is entered once and is never read back. Each call reads the settings again.
   */
  createSecret(group: string, name: string, value: string): Promise<string>
  renameSecret(id: string, name: string): void
  writeSecret(id: string, value: string): void
  deleteSecret(id: string): void
  deleteSecretGroup(group: string): void
  /** Puts a secret before another one of its group, or at the end of the group where `beforeId` is empty. */
  moveSecret(id: string, beforeId: string): void
  renameSecretGroup(group: string, name: string): void
  /** Lets one declaration of a plugin read a secret. The group is empty for a declaration with none. */
  allowPluginSecret(secretId: string, pluginId: string, name: string, group: string): void
  denyPluginSecret(secretId: string, pluginId: string, name: string, group: string): void
  /** A plugin whose last start failed, started once more. */
  retryPlugin(pluginId: string): void
  /**
   * Opens the machine's native picker for a plugin's setting of type file or folder, and
   * writes the chosen path. Resolves to an empty string where the path was written or none
   * was chosen.
   */
  choosePath(pluginId: string, key: string): Promise<string>
  /**
   * How many files the folder of a plugin holds in the shown vault, `.numen/<plugin id>/`. A window
   * that cannot delete plugin data has neither call.
   */
  getPluginData?(pluginId: string): Promise<number>
  /**
   * That folder deleted, with the text versions the plugin made there. The answer is why it was not
   * deleted, and empty where it was.
   */
  deletePluginData?(pluginId: string): Promise<string>
  /**
   * The components a plugin's frontend registered for its settings views, by view id. A view the
   * plugin declares and has not registered is absent.
   */
  getSettingsViews?(pluginId: string): ReadonlyMap<string, { readonly component: unknown }>
  readonly plugins?: Readonly<Ref<readonly PluginInfo[]>>
  readonly app?: Readonly<Ref<readonly PluginSettingInfo[]>>
  readonly catalog?: Readonly<Ref<readonly CatalogEntry[]>>
  readonly secretSections?: Readonly<Ref<readonly SecretSection[]>>
  readonly secretMenus?: Readonly<Ref<readonly SecretMenu[]>>
  readonly secretStates?: Readonly<Ref<ReadonlyMap<string, boolean>>>
  /** Reads which secrets have a value. Only the page of secrets calls it. */
  loadSecretStates?(): Promise<void>
  readonly credentialsPath?: Readonly<Ref<string>>
  readonly file: Readonly<Ref<string>>
}

/** What the settings tab holds. */
export interface SettingsTabState {
  readonly installation: Installation
  readonly initialPluginId?: Ref<string | null>
  readonly initialSearchQuery?: Ref<string>
}
