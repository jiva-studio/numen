/**
 * Whether a node in the plex nests the headings of its note under the box, and how
 * many of them stand there at once, as the settings file holds them.
 */

/** How many headings a node nests where the settings name no number. */
export const DEFAULT_HEADINGS = 6

/** Whether a node nests the headings of its note, and how many stand at once. */
export interface NestingSettings {
  readonly isNesting: boolean
  readonly headings: number
  readonly least: number
  readonly most: number
}

/** The counts offered, from one end of what the setting takes to the other. */
export const ladder = (least: number, most: number): readonly number[] =>
  Array.from({ length: Math.max(0, most - least + 1) }, (_, at) => least + at)
