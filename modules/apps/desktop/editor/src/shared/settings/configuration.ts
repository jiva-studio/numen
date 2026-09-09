/**
 * What the settings hold: one setting written where it stands, and the models a
 * setting that names one can be set to.
 */

/** One setting of the file, and what to put there. */
export interface SettingEdit {
  /** The setting, as a path through the file. */
  readonly at: readonly string[]
  /** What stands there, as JSON. */
  readonly value: string
}

/**
 * What a model's files are on this machine. A model reached over the network
 * has nothing to fetch, and where files stand says nothing about it.
 */
export type Presence = 'present' | 'not fetched' | 'nothing to fetch'

/** One model a setting that names a model can be set to. */
export interface Model {
  /** The setting it is read from. The one in force names this model there. */
  readonly namedAt: readonly string[]
  readonly name: string
  /** What is drawn on the row, and the shelf the rows around it stand under. */
  readonly title: string
  readonly shelf: string
  /** Set on the model an installation nobody has configured runs on. */
  readonly byDefault: boolean
  /** What choosing it writes. */
  readonly writes: readonly SettingEdit[]
  /** What this model's files are on this machine. */
  readonly presence: Presence
}

/** Every setting as it stands, where they stand, and the models offered. */
export interface Configuration {
  /** Every setting as JSON, the defaults under everything the file leaves out. */
  readonly written: string
  /** The file itself, absolute on this machine. */
  readonly path: string
  readonly models: readonly Model[]
}
