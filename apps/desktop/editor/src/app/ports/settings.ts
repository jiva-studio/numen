/** What the window asks of the vault about its settings. */
import type { Configuration, SettingEdit } from '@/entities/settings'

export interface SettingsPort {
  /**
   * Whether renaming either a note's title or the name of its file brings the
   * other into line, as the settings hold it.
   */
  getSyncEnabled(): Promise<boolean>
  /**
   * That setting written into the settings file. What could not be written, and
   * nothing where it was: the rename after this reads what was written.
   */
  setSyncEnabled(isEnabled: boolean): Promise<string | null>
  /** Every setting currently, and the models the settings offer. */
  getSettings(): Promise<Configuration>
  updateSettings(edits: readonly SettingEdit[]): Promise<void>
  /**
   * The key of a secret a plugin declares, stored in the system keychain and
   * never in the settings file. A machine without a keychain refuses it.
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
   * The machine's own picker put in front of the person for a plugin's setting of type file or
   * folder, and the path chosen written. The problem is why the setting does not take the path,
   * and empty where it was written.
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
