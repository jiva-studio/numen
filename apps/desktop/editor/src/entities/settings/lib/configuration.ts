/**
 * What the settings hold: one setting written at its current position, and the models a
 * setting that names one can be set to.
 */
import type { PluginSettingType } from '@numen/plugin-sdk'
export type { CompletionChoice, PluginSettingType } from '@numen/plugin-sdk'

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

/** One value an enum setting can hold, with its display title. */
export interface PluginSettingOption {
  readonly value: string
  /** Empty shows the value. */
  readonly title: string
  /** Text displayed under the setting while the value is selected. */
  readonly detail: string
}

/** A setting is shown, and handed to its plugin, while another holds one of the values listed. */
export interface PluginSettingCondition {
  readonly setting: string
  readonly is: readonly string[]
}

/** The options of a setting under each value of another setting; the first of a list is its default. */
export interface PluginSettingOptionsBy {
  readonly setting: string
  readonly options: Readonly<Record<string, readonly PluginSettingOption[]>>
}

/** A secret, such as an API key, that a plugin requests the host to store. */
export interface PluginSecret {
  /** What the plugin reads the key under. */
  readonly name: string
  readonly title: string
  readonly description: string
  readonly when?: PluginSettingCondition | undefined
}

/** Declarative setting schema and metadata exposed by a plugin. */
export interface PluginSettingInfo {
  readonly key: string
  readonly type: PluginSettingType
  readonly title: string
  readonly description: string
  readonly default: unknown
  readonly palette?: boolean | undefined
  readonly icon?: string | undefined
  readonly options?: readonly PluginSettingOption[] | undefined
  readonly optionsBy?: PluginSettingOptionsBy | undefined
  readonly when?: PluginSettingCondition | undefined
  /** The file extensions a setting of type file takes, each with its leading dot. Empty takes any file. */
  readonly accept?: readonly string[] | undefined
  readonly min?: number | undefined
  readonly max?: number | undefined
  readonly step?: number | undefined
  /** The agent program whose models the setting offers; empty when the user types the value. */
  readonly modelsOf?: string | undefined
}

/**
 * Whether the user permits a plugin to call a model: `none` is a plugin without
 * the ai:complete permission, `requested` is a plugin that requested it since the
 * host started and has received no answer.
 */
export type Consent = 'none' | 'pending' | 'requested' | 'allowed' | 'denied'

/** Why the host suspends a plugin after it failed to start, and when it gave the plugin up. */
export interface PluginFailure {
  /** A short UI label, such as "Failed to start". */
  readonly reason: string
  /** The underlying error, shown on request. Empty when there is none. */
  readonly detail: string
  /** When the host gave the plugin up, in milliseconds since the Unix epoch. */
  readonly failedAt: number
}

/** Information about an installed and active plugin, including declarative settings. */
export interface PluginInfo {
  readonly id: string
  readonly name: string
  readonly version: string
  readonly description: string
  readonly isActive: boolean
  readonly settings: readonly PluginSettingInfo[]
  readonly capabilities: readonly string[]
  /** The secrets the plugin declares, and the names of those that have a key stored. */
  readonly secrets: readonly PluginSecret[]
  readonly storedSecrets: readonly string[]
  readonly consent: Consent
  /** Whether the plugin declares a frontend the window loads. */
  readonly hasFrontend: boolean
  /** The readers the plugin declares, each with the extensions of the files it opens. */
  readonly readers?: readonly PluginReader[]
  /** The views the plugin declares for its settings form, each placed after the setting named by `after`. */
  readonly settingsViews?: readonly PluginSettingsViewInfo[]
  /** Set while the host suspends the plugin. */
  readonly failure?: PluginFailure | undefined
}

/** A view a plugin's frontend registers under `id` and the settings form mounts. */
export interface PluginSettingsViewInfo {
  readonly id: string
  readonly title: string
  /** The key of the setting the view follows; empty places the view at the top of the form. */
  readonly after: string
}

/** A reader of a plugin and the extensions, lowercase and without the dot, of the files it opens. */
export interface PluginReader {
  readonly id: string
  readonly extensions: readonly string[]
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
