import type {
  CommandRegistration,
  ReaderRegistration,
  SettingsViewRegistration,
  TabKindRegistration,
  VaultChangeEvent,
} from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import type { PluginParts } from '../host'
import type { SlotHost } from '@numen/plugin-sdk'
import type { SlotViewRegistry } from '../slot-views'
import type { KeybindingRegistry, KeybindingSource } from '../keybindings'
import type { CommandApi } from './commandApi'
import type { ContextSettingHost, PluginContextHost } from './contextHostTypes'

/** How a plugin is created, marked and taken away, and the settings it reads. */
export interface PluginLifecycle {
  createContext(pluginId: string, manifest?: Manifest): PluginParts
  markPluginStarting(pluginId: string): void
  markPluginStarted(pluginId: string): void
  registerManifest(pluginId: string, manifest: Manifest): void
  disposePlugin(pluginId: string): void
  disposeAll(): void
  readonly getSetting: ContextSettingHost['getSetting']
  readonly watchSetting: ContextSettingHost['watchSetting']
}

/** The keybindings of the application and of the plugins, by owner. */
export interface KeybindingApi {
  readonly registry: KeybindingRegistry
  registerKeybindings(pluginId: string, sources: readonly KeybindingSource[]): void
  unregisterKeybindings(pluginId: string): void
  unregisterAllKeybindings(): void
  /** The key of a command as declared, such as `mod+p`, for a hint; undefined where none is bound. */
  readonly getKeyFor: KeybindingRegistry['getKeyFor']
}

/** What the plugins registered, for the windows that list it. */
export interface RegisteredApi {
  /** The settings views a plugin registered, by view id; reactive. */
  getRegisteredSettingsViews(pluginId: string): ReadonlyMap<string, SettingsViewRegistration>
  /**
   * The settings views of a plugin for its settings page, by view id; reactive. A plugin that
   * declares settings views and has registered none starts, and its views follow once it has.
   */
  getSettingsViews(pluginId: string): ReadonlyMap<string, SettingsViewRegistration>
  getRegisteredReaders(): ReaderRegistration[]
  getRegisteredTabKinds(): TabKindRegistration[]
  getRegisteredCommands(): CommandRegistration[]
}

export interface PluginHost {
  readonly lifecycle: PluginLifecycle
  readonly commands: CommandApi
  readonly keybindings: KeybindingApi
  readonly registered: RegisteredApi
  /** The entries of the slots of the window, as their owners read them. */
  readonly slots: SlotHost
  readonly slotViews: SlotViewRegistry
  readonly contextHost: PluginContextHost
  /** A change to the vault, delivered to every plugin that subscribed to vault changes. */
  notifyVaultChange(event: VaultChangeEvent): void
}
