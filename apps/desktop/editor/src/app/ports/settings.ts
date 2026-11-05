/** The vault settings operations the window uses. */
import type { Configuration, SettingEdit } from '@/entities/settings'

export interface SettingsPort {
  /** Every setting currently, and the models the settings offer. */
  getSettings(): Promise<Configuration>
  updateSettings(edits: readonly SettingEdit[]): Promise<void>
  /**
   * The settings that changed, until the signal aborts: a write by this window, another window, a
   * plugin or the palette, and an edit of the file by hand. Each batch holds the settings that
   * changed together, named by their declared path, with the value each now holds as JSON. The
   * stream opens with an empty batch, which the window ignores.
   */
  watchSettings(signal: AbortSignal): AsyncIterable<readonly SettingEdit[]>
  /**
   * The key of a secret declared by a plugin, stored in the system keychain and
   * never in the settings file. A machine without a keychain rejects it.
   */
  writePluginSecret(pluginId: string, name: string, key: string): Promise<void>
  /** The key of a secret a plugin declares, removed. */
  deletePluginSecret(pluginId: string, name: string): Promise<void>
  /**
   * A plugin the host suspends, started once more. The answer is why it failed
   * again, and empty where it started.
   */
  retryPlugin(pluginId: string): Promise<string>
  /**
   * Opens the system picker for a plugin setting of type file or folder, and
   * writes the selected path. The returned problem states why the setting does not
   * accept the path, and is empty when the path was written.
   */
  choosePluginPath(
    pluginId: string,
    key: string,
  ): Promise<{ readonly path: string; readonly isChosen: boolean; readonly problem: string }>
  /** How many files the folder of a plugin holds in the shown vault, `.numen/<plugin id>/`. */
  getPluginData(pluginId: string): Promise<number>
  /** The folder of a plugin in the shown vault deleted, with the text versions the plugin made there. */
  deletePluginData(pluginId: string): Promise<void>
}
