/**
 * The plugins as the manager shows them: the settings of a plugin decoded, and the health items the
 * host reports for it.
 */

/** A value a setting holds: anything that survives JSON.stringify and JSON.parse unchanged. */
export type SettingValue = string | number | boolean | null | SettingList | SettingObject

/** The list a setting value holds. */
interface SettingList extends ReadonlyArray<SettingValue> {
  readonly length: number
}

/** The object a setting value holds. */
interface SettingObject {
  readonly [key: string]: SettingValue
}

/** Declarative setting value types exposed by a plugin. */
export type PluginSettingType =
  'boolean' | 'string' | 'number' | 'enum' | 'range' | 'file' | 'connections'

/** One value an enum setting can hold, with its display title. */
export interface PluginSettingOption {
  readonly value: string
  readonly title: string
  readonly detail: string
  readonly when?: string | undefined
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

/** Declarative setting schema and metadata exposed by a plugin. */
export interface PluginSettingInfo {
  readonly key: string
  readonly type: PluginSettingType
  readonly title: string
  readonly description: string
  readonly default: SettingValue | undefined
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
  /** The operations a setting of type connections is used for. */
  readonly operations?: readonly string[] | undefined
  readonly activeOptions?: readonly PluginSettingOption[] | undefined
  readonly isShown?: boolean | undefined
  readonly value?: SettingValue | undefined
}

/**
 * One concern of a plugin that does not work, as the manager draws it. An empty action label offers
 * no button. The key `start` is the start of the plugin that failed, which the host holds suspended.
 */
export interface PluginHealth {
  readonly key: string
  readonly severity: 'warning' | 'error'
  readonly text: string
  readonly detail: string
  readonly actionLabel: string
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
  readonly readers?: readonly PluginReader[]
  readonly settingsViews?: readonly PluginSettingsViewInfo[]
  readonly contributions?: readonly PluginContribution[]
}

/** A section of a manifest, named after its key, whose entries the plugin page lists. */
export type PluginContributionKind =
  | 'commands'
  | 'keybindings'
  | 'tabKinds'
  | 'readers'
  | 'handlers'
  | 'triggers'
  | 'textKinds'
  | 'noteTypes'
  | 'themes'
  | 'slots'
  | 'listeners'
  | 'viewCapabilities'
  | 'dropHandlers'
  | 'frameOrigins'
  | 'connectionTypes'
  | 'operations'
  | 'permissions'
  | 'dependencies'
  | 'services'
  | 'capabilities'
  | 'activationEvents'

/** One section of a manifest with its entries, in the order the manifest declares them. */
export interface PluginContribution {
  readonly kind: PluginContributionKind
  readonly items: readonly PluginContributionItem[]
}

/**
 * One entry of a section: its title, or its id where it has none; its id or target as the manifest
 * writes it; and for a command or a keybinding the shortcut, such as `mod+shift+a`.
 */
export interface PluginContributionItem {
  readonly text: string
  readonly detail: string
  readonly keys: string
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
