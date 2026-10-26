/** What the window asks of the vault about its settings. */
import type { Configuration, SettingEdit } from '@/entities/settings'
import type { NestingSettings } from '@/entities/settings'

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
  /**
   * Whether a node in the plex nests the headings of its note under the box, and
   * how many of them stand there at once, as the settings hold them.
   */
  getNestingSettings(): Promise<NestingSettings>
  /**
   * Those settings written into the settings file. What could not be written,
   * and nothing where it was. A count left out stands as it is.
   */
  setNestingSettings(isNesting: boolean, headings?: number): Promise<string | null>
  /** Every setting currently, and the models the settings offer. */
  getSettings(): Promise<Configuration>
  updateSettings(edits: readonly SettingEdit[]): Promise<void>
  /** The settings file as its person wrote it, and at its current position. */
  getSettingsFile(): Promise<{ readonly written: string; readonly path: string }>
  saveSettingsFile(
    content: string,
    expectedVersion: string | null,
  ): Promise<{ readonly isChanged: boolean }>
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
}
