/**
 * What the settings hold: one setting written at its current position, and the models a
 * setting that names one can be set to.
 */
import type { PluginInfo, PluginSettingInfo } from '@numen/plugin-manager'
export type {
  PluginInfo,
  SecretDeclaration,
  PluginSettingCondition,
  PluginSettingInfo,
  PluginSettingOption,
  PluginSettingsViewInfo,
} from '@numen/plugin-manager'

/** One declaration of a plugin that may read a secret. */
interface DeclarationKey {
  readonly pluginId: string
  readonly name: string
  readonly group: string
}

/** One secret as the window holds it. A value is never part of it. */
export interface Secret {
  readonly id: string
  readonly name: string
  readonly group: string
  readonly allowed: readonly DeclarationKey[]
  readonly hasValue?: boolean
}

/** The secrets of one group in priority order, as the core orders the page of secrets. */
export interface SecretSection {
  readonly group: string
  readonly isAsked: boolean
  readonly secrets: readonly Secret[]
}

/** The secrets the menu of one secret declaration of a plugin offers, in order. */
export interface SecretMenu {
  readonly pluginId: string
  readonly name: string
  readonly group: string
  readonly secrets: readonly Secret[]
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

/** Every setting currently in effect, and the models offered. */
export interface Configuration {
  readonly jsonContent?: string | undefined
  readonly written: string
  readonly path: string
  readonly models: readonly Model[]
  readonly plugins?: readonly PluginInfo[] | undefined
  readonly app?: readonly PluginSettingInfo[] | undefined
  readonly secretSections?: readonly SecretSection[] | undefined
  readonly secretMenus?: readonly SecretMenu[] | undefined
  readonly credentialsPath?: string | undefined
}
