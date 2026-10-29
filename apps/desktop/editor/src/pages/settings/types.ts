/**
 * The settings tab, and what it holds.
 *
 * It holds nothing of its own: the window already keeps every setting it can
 * write, and this is a second way to the same values. A row here and the
 * command of the same name in the palette go through one piece of code.
 */
import type { Ref } from 'vue'
import type { Model, PluginInfo, SettingEdit } from '@/entities/settings'
import type { Mode, Ranges, Sizes, Theme } from '@/entities/settings'

/** What this installation is configured as, as the window already holds it. */
export interface Installation {
  /** Every theme there is, and the one the settings name. */
  readonly themes: Readonly<Ref<readonly Theme[]>>
  readonly applied: Readonly<Ref<string>>
  readonly mode: Readonly<Ref<Mode>>
  /** Whether the active theme declares light and dark itself. */
  readonly isPinned: Readonly<Ref<boolean>>
  readonly sizes: Readonly<Ref<Sizes>>
  readonly bounds: Readonly<Ref<Ranges>>
  /**
   * Selects a value for one of the four appearance settings, by the identity the palette
   * offers it under. The theme, the light/dark half of the pair and the two sizes are each
   * named differently, so none of the four is typed manually.
   */
  choose(item: string): void
  /** Whether a note's title and the name of its file are kept as one name. */
  readonly isSyncing: Ref<boolean>
  /**
   * names one can be set to, and settings written where they stand.
   */
  getSetting(at: readonly string[]): unknown
  getModels(at: readonly string[]): readonly Model[]
  write(content: readonly SettingEdit[]): void
  /**
   * The key of a secret a plugin declares, stored or removed. The window is informed only
   * which secrets have a key, under each plugin's `storedSecrets`.
   */
  writeSecret(pluginId: string, name: string, key: string): void
  deleteSecret(pluginId: string, name: string): void
  /** A plugin the host suspends, started once more. */
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
  /** The file the settings stand in, absolute on this machine. */
  readonly file: Readonly<Ref<string>>
}

/** What the settings tab holds. */
export interface SettingsTabState {
  readonly installation: Installation
  readonly initialPluginId?: Ref<string | null>
  readonly initialSearchQuery?: Ref<string>
}
