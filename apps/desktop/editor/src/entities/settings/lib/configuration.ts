/**
 * What the settings hold: one setting written at its current position, and the models a
 * setting that names one can be set to.
 */
import type { PluginInfo, PluginSettingInfo } from '@numen/plugin-manager'
export type {
  PluginInfo,
  PluginReader,
  PluginSecret,
  PluginSettingCondition,
  PluginSettingInfo,
  PluginSettingOption,
  PluginSettingOptionsBy,
  PluginSettingsViewInfo,
} from '@numen/plugin-manager'
export type { PluginSettingType } from '@numen/core-client'

/** One declaration of a plugin that may read a secret. */
export interface SecretPermission {
  readonly pluginId: string
  /** The name of the declaration in the manifest of the plugin. */
  readonly name: string
  /** The group of the declaration, empty for a declaration with none. */
  readonly group: string
}

/** One secret as the window holds it. A value is never part of it. */
export interface Secret {
  readonly id: string
  readonly name: string
  readonly group: string
  /** The declarations that may read the secret. */
  readonly allowed: readonly SecretPermission[]
  /** Whether the value is on this machine. */
  readonly hasValue: boolean
}

/** One setting of the configuration file, and the JSON value to set. */
export interface SettingEdit {
  readonly at: readonly string[]
  readonly value: string
}

/**
 * Whether a model's files are on this machine. A model reached over the network
 * has nothing to fetch, so file presence does not apply to it.
 */
export type Presence = 'present' | 'not fetched' | 'nothing to fetch'

/** One AI model option a configuration setting can select. */
export interface Model {
  readonly namedAt: readonly string[]
  readonly name: string
  readonly title: string
  readonly group: string
  readonly isDefault: boolean
  readonly edits?: readonly SettingEdit[]
  readonly writes: readonly SettingEdit[]
  readonly presence: Presence
  readonly reasoningLevels?: readonly { readonly name: string; readonly title: string }[]
  readonly defaultReasoning?: string
}

/** Every setting currently, where they stand, and the models offered. */
export interface Configuration {
  readonly jsonContent?: string | undefined
  readonly written: string
  readonly path: string
  readonly models: readonly Model[]
  readonly plugins?: readonly PluginInfo[] | undefined
  readonly app?: readonly PluginSettingInfo[] | undefined
  /** Every secret in priority order, without its value. */
  readonly secrets?: readonly Secret[] | undefined
  /** The file the secret values are kept in where the system keychain is not in use, else empty. */
  readonly credentialsPath?: string | undefined
}
