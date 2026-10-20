/**
 * What the settings hold: one setting written where it stands, and the models a
 * setting that names one can be set to.
 */

/** One setting of the configuration file, and the JSON value to set. */
export interface SettingEdit {
  /** The setting key path through the JSON file (e.g. ['indexing', 'provider']). */
  readonly at: readonly string[]
  /** What stands there, as serialized JSON. */
  readonly value: string
}

/**
 * What a model's files are on this machine. A model reached over the network
 * has nothing to fetch, and where files stand says nothing about it.
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
  /** The depths of reasoning the model can be asked to work at, from the least to the most. */
  readonly reasoningLevels?: readonly { readonly name: string; readonly title: string }[]
  /** The level the model works at unasked, empty where it names none. */
  readonly defaultReasoning?: string
}

/** Declarative setting schema and metadata exposed by a plugin. */
export interface PluginSettingInfo {
  readonly key: string
  readonly type: 'boolean' | 'string' | 'number' | 'enum' | 'range' | 'agent'
  readonly title: string
  readonly description: string
  readonly default: unknown
  readonly palette?: boolean | undefined
  readonly options?: readonly string[] | undefined
  readonly min?: number | undefined
  readonly max?: number | undefined
  readonly step?: number | undefined
  /** The agent program whose models the setting is offered; empty where a person types its value. */
  readonly modelsOf?: string | undefined
}

/**
 * Whether the person lets a plugin ask a model: `none` is a plugin that does
 * not hold the ai:complete permission, `asked` one that asked since the host
 * started and has had no answer.
 */
export type Consent = 'none' | 'unanswered' | 'asked' | 'allowed' | 'refused'

/** Why the host suspends a plugin after it failed to start, and when it gave the plugin up. */
export interface PluginFailure {
  /** The reason, in words a person reads. */
  readonly reason: string
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
  /** The names of the secrets the plugin declares, and those that have a key stored. */
  readonly secrets: readonly string[]
  readonly storedSecrets: readonly string[]
  readonly consent: Consent
  /** Set while the host suspends the plugin. */
  readonly failure?: PluginFailure | undefined
}

/** The agent program and the model a setting of type `agent` holds. Empty is the default. */
export interface CompletionChoice {
  readonly agent: string
  readonly model: string
}

/** Every setting as it stands, where they stand, and the models offered. */
export interface Configuration {
  /** Serialized settings JSON content. */
  readonly jsonContent?: string | undefined
  readonly written: string
  /** Absolute path to the settings file on disk. */
  readonly path: string
  readonly models: readonly Model[]
  readonly plugins?: readonly PluginInfo[] | undefined
}
