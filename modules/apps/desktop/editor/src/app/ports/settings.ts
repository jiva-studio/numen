/** What the window asks of the vault about its settings. */
import type { Configuration, SettingEdit } from '@/entities/settings'
import type { HangingSettings } from '@/entities/settings'
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
  setSyncEnabled(kept: boolean): Promise<string | null>
  /**
   * Whether a node in the plex hangs the parts of its note under the box, and
   * how many of them stand there at once, as the settings hold them.
   */
  getHangingSettings(): Promise<HangingSettings>
  /**
   * Those settings written into the settings file. What could not be written,
   * and nothing where it was. A count left out stands as it is.
   */
  setHangingSettings(hangs: boolean, parts?: number): Promise<string | null>
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
  updateSettings(written: readonly SettingEdit[]): Promise<void>
  /** The settings file as its person wrote it, and where it stands. */
  getSettingsFile(): Promise<{ readonly written: string; readonly path: string }>
  /**
   * The settings file replaced whole, with the bytes as they were typed. A file
   * the settings could not be read out of is refused, and what the file holds
   * is unchanged.
   *
   * Seen is the file as it was last read, and a file standing at anything else
   * is answered `changed` with nothing written. Nothing seen writes over
   * whatever the file holds.
   */
  saveSettingsFile(
    written: string,
    seen: string | null,
  ): Promise<{ readonly changed: boolean }>
}
