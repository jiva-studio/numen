/** What the window asks of the vault about its settings. */
import type { Configuration, SettingEdit } from '@/entities/settings'
import type { NestingSettings } from '@/entities/settings'
import type { ReviewSettings } from '@/entities/settings'

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
  /**
   * The hour a day of review begins at, on the clock on the wall, written as
   * `04:00`, and how late in the day the vault takes one. An hour past that is
   * refused.
   */
  getReviewSettings(): Promise<ReviewSettings>
  /**
   * That hour written into the settings file. What could not be written, and
   * nothing where it was.
   */
  setReviewSettings(starts: string): Promise<string | null>
  /** Every setting as it stands, and the models the settings offer. */
  getSettings(): Promise<Configuration>
  /**
   * Settings written into the settings file, together or not at all. A value
   * the settings could not be read out of again is refused, and what the file
   * holds is unchanged.
   */
  updateSettings(edits: readonly SettingEdit[]): Promise<void>
  /** The settings file as its person wrote it, and where it stands. */
  getSettingsFile(): Promise<{ readonly written: string; readonly path: string }>
  /**
   * The settings file replaced whole, with the bytes as they were typed. A file
   * the settings could not be read out of is refused, and what the file holds
   * is unchanged.
   *
   * Expected version is the file as it was last read, and a file standing at anything else
   * is answered `changed` with nothing written.
   */
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
}
