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
  /** Whether the theme worn declares light and dark itself. */
  readonly isPinned: Readonly<Ref<boolean>>
  readonly sizes: Readonly<Ref<Sizes>>
  readonly bounds: Readonly<Ref<Ranges>>
  /**
   * A value of one of the four appearance settings chosen, by the identity the
   * palette offers it under. The theme, the half of the pair and the two sizes
   * are each named a different way, so none of the four is written by hand.
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
   * The key of a secret a plugin declares, stored or removed. The window is
   * told only which secrets have a key, under each plugin's `storedSecrets`.
   */
  writeSecret(pluginId: string, name: string, key: string): void
  deleteSecret(pluginId: string, name: string): void
  /** A plugin the host suspends, started once more. */
  retryPlugin(pluginId: string): void
  /**
   * The machine's own picker put in front of the person for a plugin's setting of type file or
   * folder, and the path chosen written. The answer is why the setting does not take the path,
   * and empty where it was written or none was chosen.
   */
  choosePath(pluginId: string, key: string): Promise<string>
  readonly plugins?: Readonly<Ref<readonly PluginInfo[]>>
  /** The file the settings stand in, absolute on this machine. */
  readonly file: Readonly<Ref<string>>
  /** That file opened whole, in a tab of its own. */
  openFile(): void
}

/** What the settings tab holds. */
export interface SettingsTabState {
  readonly installation: Installation
  readonly initialPluginId?: Ref<string | null>
}
