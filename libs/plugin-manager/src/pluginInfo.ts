/**
 * The plugins as the manager shows them: the settings of a plugin decoded, and the state the host
 * reports for it.
 */

/** Declarative setting value types exposed by a plugin. */
export type PluginSettingType = 'boolean' | 'string' | 'number' | 'enum' | 'range' | 'file'

/** One value an enum setting can hold, with its display title. */
export interface PluginSettingOption {
  readonly value: string
  readonly title: string
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
  readonly name: string
  readonly title: string
  readonly description: string
  /** Where the person finds the secrets for this declaration; empty for none. A hint, not a limit. */
  readonly group: string
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
  readonly accept?: readonly string[] | undefined
  readonly multiple?: boolean | undefined
  readonly min?: number | undefined
  readonly max?: number | undefined
  readonly step?: number | undefined
  readonly preview?: boolean | undefined
  readonly optionsFrom?: string | undefined
  readonly activeOptions?: readonly PluginSettingOption[] | undefined
  readonly isShown?: boolean | undefined
  readonly value?: unknown
}

/** Why the host suspends a plugin after it failed to start, and when it gave the plugin up. */
export interface PluginFailure {
  readonly reason: string
  readonly detail: string
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
  readonly hasBackend: boolean
  readonly settings: readonly PluginSettingInfo[]
  readonly capabilities: readonly string[]
  /** The secrets the plugin declares. */
  readonly secrets: readonly PluginSecret[]
  /** The readers the plugin declares, each with the extensions of the files it opens. */
  readonly readers?: readonly PluginReader[]
  readonly settingsViews?: readonly PluginSettingsViewInfo[]
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
