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
  readonly createContext: (pluginId: string, manifest?: Manifest) => PluginParts
  readonly markPluginStarting: (pluginId: string) => void
  readonly markPluginStarted: (pluginId: string) => void
  readonly registerManifest: (pluginId: string, manifest: Manifest) => void
  readonly disposePlugin: (pluginId: string) => void
  readonly disposeAll: () => void
  readonly getSetting: ContextSettingHost['getSetting']
  readonly watchSetting: ContextSettingHost['watchSetting']
}

/** The keybindings of the application and of the plugins, by owner. */
export interface KeybindingApi {
  readonly registry: KeybindingRegistry
  readonly registerKeybindings: (pluginId: string, sources: readonly KeybindingSource[]) => void
  readonly unregisterKeybindings: (pluginId: string) => void
  readonly unregisterAllKeybindings: () => void
  readonly getKeyFor: KeybindingRegistry['getKeyFor']
}

/** What the plugins registered, for the windows that list it. */
export interface RegisteredApi {
  /** The settings views a plugin registered, by view id; reactive. */
  readonly getRegisteredSettingsViews: (
    pluginId: string,
  ) => ReadonlyMap<string, SettingsViewRegistration>
  /**
   * The settings views of a plugin for its settings page, by view id; reactive. A plugin that
   * declares settings views and has registered none starts, and its views follow once it has.
   */
  readonly getSettingsViews: (pluginId: string) => ReadonlyMap<string, SettingsViewRegistration>
  readonly getRegisteredReaders: () => ReaderRegistration[]
  readonly getRegisteredTabKinds: () => TabKindRegistration[]
  readonly getRegisteredCommands: () => CommandRegistration[]
}

export interface PluginHost {
  readonly lifecycle: PluginLifecycle
  readonly commands: CommandApi
  readonly keybindings: KeybindingApi
  readonly registered: RegisteredApi
  readonly slots: SlotHost
  readonly slotViews: SlotViewRegistry
  readonly contextHost: PluginContextHost
  /** A change to the vault, delivered to every plugin that subscribed to vault changes. */
  readonly notifyVaultChange: (event: VaultChangeEvent) => void
}
