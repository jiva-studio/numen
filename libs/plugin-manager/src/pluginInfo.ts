/**
 * The plugins as the manager shows them: the settings of a plugin decoded, and the state the host
 * reports for it.
 */

/** Declarative setting value types exposed by a plugin. */
export type PluginSettingType = 'boolean' | 'string' | 'number' | 'enum' | 'range' | 'file'

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
  /** Whether a setting of type file holds a list of paths, one per line. */
  readonly multiple?: boolean | undefined
  readonly min?: number | undefined
  readonly max?: number | undefined
  readonly step?: number | undefined
  /** Whether the window applies the value while its control is focused, and restores it on leaving. */
  readonly preview?: boolean | undefined
  /** The registry the options are listed from, such as the installed themes. */
  readonly optionsFrom?: string | undefined
  /** The options in force under the values in force, resolved by the core. */
  readonly activeOptions?: readonly PluginSettingOption[] | undefined
  /** Whether the condition of the setting holds under the values in force, resolved by the core. */
  readonly isShown?: boolean | undefined
  /** The value in force, resolved by the core: the stored value, else the default, else the first option. */
  readonly value?: unknown
}

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
  readonly hasFrontend: boolean
  /** Whether the plugin declares a backend the host starts. */
  readonly hasBackend: boolean
  readonly settings: readonly PluginSettingInfo[]
  readonly capabilities: readonly string[]
  /** The secrets the plugin declares, and the names of those that have a key stored. */
  readonly secrets: readonly PluginSecret[]
  readonly storedSecrets: readonly string[]
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
}

/** A reader of a plugin and the extensions, lowercase and without the dot, of the files it opens. */
export interface PluginReader {
  readonly id: string
  readonly extensions: readonly string[]
}
