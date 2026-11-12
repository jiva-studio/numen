/**
 * The settings tab, and its contents.
 *
 * It holds nothing of its own: the window already keeps every setting it can
 * write, and this is a second way to the same values. A row here and the
 * command of the same name in the palette go through one piece of code.
 */
import type { Ref } from 'vue'
import type { CatalogEntry } from '@/entities/plugin'
import type {
  Model,
  PluginInfo,
  PluginSettingInfo,
  SecretMenu,
  SecretSection,
  SettingEdit,
} from '@/entities/settings'
import type { JsonValue } from '@/shared/api/json'
import type { Component } from 'vue'

/** What this installation is configured as, as the window already holds it. */
export interface Installation {
  readonly settings: InstallationSettings
  readonly plugin: InstallationPlugin
  readonly secrets: InstallationSecrets
}

/**
 * The settings of the installation, read and written in place. `getPlatform` answers the operating
 * system of the window: darwin, windows or linux; the `app.platform` of conditions.
 */
/** The component a plugin registered for a view of its settings page. */
export interface ViewComponent {
  readonly component: Component
}

export interface InstallationSettings {
  /**
   * names one can be set to, and settings written in place.
   */
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
  getModels(at: readonly string[]): readonly Model[]
  getPlatform(): string
  write(content: readonly SettingEdit[]): void
  readonly file: Readonly<Ref<string>>
}

/**
 * The plugins of the installation, what they declare and what the window does for them.
 * `getSettingsViews` answers the components a plugin's frontend registered for its settings
 * views, by view id; a view the plugin declares and has not registered is absent.
 */
export interface InstallationPlugin {
  readonly plugins: Readonly<Ref<readonly PluginInfo[]>>
  readonly app: Readonly<Ref<readonly PluginSettingInfo[]>>
  readonly catalog: Readonly<Ref<readonly CatalogEntry[]>>
  /** A plugin whose last start failed, started once more. */
  retryPlugin(pluginId: string): void
  /**
   * Opens the machine's native picker for a plugin's setting of type file or folder, and
   * writes the chosen path. Resolves to an empty string where the path was written or none
   * was chosen.
   */
  choosePath(pluginId: string, key: string): Promise<string>
  getSettingsViews(pluginId: string): ReadonlyMap<string, ViewComponent>
  readonly files: InstallationPluginFiles
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
 * The secrets, each a name in a group with the plugin declarations that may read it. A value
 * is entered once and is never read back. Each call reads the settings again.
 */
export interface InstallationSecrets {
  readonly sections: Readonly<Ref<readonly SecretSection[]>>
  readonly menus: Readonly<Ref<readonly SecretMenu[]>>
  readonly states: Readonly<Ref<ReadonlyMap<string, boolean>>>
  /** Reads which secrets have a value. Only the page of secrets calls it. */
  loadStates(): Promise<void>
  readonly credentialsPath: Readonly<Ref<string>>
  readonly edit: {
    createSecret(group: string, name: string, value: string): Promise<string>
    renameSecret(id: string, name: string): void
    writeSecret(id: string, value: string): void
    deleteSecret(id: string): void
    deleteSecretGroup(group: string): void
    /** Puts a secret before another one of its group, or at the end of the group where `beforeId` is empty. */
    moveSecret(id: string, beforeId: string): void
    renameSecretGroup(group: string, name: string): void
  }
  readonly access: {
    /** Lets one declaration of a plugin read a secret. The group is empty for a declaration with none. */
    allowPluginSecret(secretId: string, pluginId: string, name: string, group: string): void
    denyPluginSecret(secretId: string, pluginId: string, name: string, group: string): void
  }
}

/** One of the choices a select offers. */
export interface SelectChoice {
  readonly id: string
  readonly text: string
  readonly detail?: string
  readonly group?: string
  readonly isDisabled?: boolean
}

/** What the fields of one plugin ask of the plugins of the installation. */
export type PluginFieldsHost = Pick<InstallationPlugin, 'choosePath' | 'getSettingsViews'>

/** The secrets a field offers to allow and deny, and how they are allowed. */
export type SecretChoices = Pick<InstallationSecrets, 'sections' | 'menus' | 'access'>

/** What the settings tab holds. */
export interface SettingsTabState {
  readonly installation: Installation
  readonly initialPluginId: Ref<string | null>
  readonly initialSearchQuery: Ref<string>
}
