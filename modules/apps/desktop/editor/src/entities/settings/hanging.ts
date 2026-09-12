/**
 * Whether a node in the plex hangs the parts of its note under the box, and how
 * many of them stand there at once, as the settings file holds them.
 */

/** How many a node hangs where the settings name no number. */
export const DEFAULT_PARTS = 6

/** Whether a node hangs the parts of its note, and how many stand at once. */
export interface HangingSettings {
  readonly hangs: boolean
  readonly parts: number
  /** How many the vault takes, at each end. A count outside them is refused. */
  readonly least: number
  readonly most: number
}

/** The counts offered, from one end of what the setting takes to the other. */
export const ladder = (least: number, most: number): readonly number[] =>
  Array.from({ length: Math.max(0, most - least + 1) }, (_, at) => least + at)
