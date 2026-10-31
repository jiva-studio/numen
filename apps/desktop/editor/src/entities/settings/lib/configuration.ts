/**
 * What the settings hold: one setting written at its current position, and the models a
 * setting that names one can be set to.
 */
import type { PluginInfo } from '@numen/plugin-sdk'
export type {
  CompletionChoice,
  Consent,
  PluginFailure,
  PluginInfo,
  PluginReader,
  PluginSecret,
  PluginSettingCondition,
  PluginSettingInfo,
  PluginSettingOption,
  PluginSettingOptionsBy,
  PluginSettingsViewInfo,
  PluginSettingType,
} from '@numen/plugin-sdk'

/** One setting of the configuration file, and the JSON value to set. */
export interface SettingEdit {
  /** The setting key path through the JSON file (e.g. ['indexing', 'provider']). */
  readonly at: readonly string[]
  /** The value at that path, as serialized JSON. */
  readonly value: string
}

/**
 * Whether a model's files are on this machine. A model reached over the network
 * has nothing to fetch, so file presence does not apply to it.
 */
export type Presence = 'present' | 'not fetched' | 'nothing to fetch'

/** One AI model option a configuration setting can select. */
export interface Model {
  /** The setting path it is read from. */
  readonly namedAt: readonly string[]
  /** Machine-readable name of the model. */
  readonly name: string
  /** Display title shown in settings rows. */
  readonly title: string
  /** Grouping category/group for settings navigation. */
  readonly group: string
  /** Whether this is the default model for fresh installations. */
  readonly isDefault: boolean
  /** Setting edits applied when this model is selected. */
  readonly edits?: readonly SettingEdit[]
  readonly writes: readonly SettingEdit[]
  /** Availability of model files on the local machine. */
  readonly presence: Presence
  /** The reasoning levels the model supports, from lowest to highest. */
  readonly reasoningLevels?: readonly { readonly name: string; readonly title: string }[]
  /** The level the model works at unasked, empty where it names none. */
  readonly defaultReasoning?: string
}

/** Every setting currently, where they stand, and the models offered. */
export interface Configuration {
  /** Serialized settings JSON content. */
  readonly jsonContent?: string | undefined
  readonly written: string
  /** Absolute path to the settings file on disk. */
  readonly path: string
  readonly models: readonly Model[]
  readonly plugins?: readonly PluginInfo[] | undefined
}
