/** What the settings hold: one setting written at its current position. */
import type { PluginInfo, PluginSettingInfo } from '@numen/plugin-manager'
export type {
  PluginContribution,
  PluginContributionKind,
  PluginInfo,
  PluginSettingCondition,
  PluginSettingInfo,
  PluginSettingOption,
  PluginSettingsViewInfo,
} from '@numen/plugin-manager'

/** One setting of the configuration file, and the JSON value to set. */
export interface SettingEdit {
  readonly at: readonly string[]
  readonly value: string
}

/** Every setting currently in effect. */
export interface Configuration {
  readonly jsonContent?: string | undefined
  readonly written: string
  readonly path: string
  readonly plugins?: readonly PluginInfo[] | undefined
  readonly app?: readonly PluginSettingInfo[] | undefined
}
