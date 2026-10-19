/** What the settings file holds, and how the window is drawn from it. */
export {
  DESIGNED,
  isInBounds,
  ladder,
  MODES,
  EMPTY_RANGE,
  parseSize,
  SCHEMES,
} from './lib/appearance'
export type {
  Configuration,
  Model,
  Presence,
  SettingEdit,
  PluginInfo,
  PluginModel,
  PluginSettingInfo,
  Consent,
  CompletionChoice,
} from './lib/configuration'
export { DEFAULT_HEADINGS, ladder as nestingLadder } from './lib/nesting'
export type { NestingSettings } from './lib/nesting'
export { DEFAULT_STARTS, createReviewSetting } from './model/review'
export type { ReviewSettings } from './model/review'
export { getSettingAt, createSettingsStore } from './model/store'
export { themes } from './api/theme'
export type { Appearance, Bounds, Mode, Ranges, Scales, Sizes, Theme, Themes } from './api/theme'
export { createSettingFields } from './lib/fields'
export type { SettingFields, SettingSource } from './lib/fields'
export { write } from './lib/write'
export { createSecretStore } from './model/secrets'
export type { SecretDeps, SecretOptions, SecretWords } from './model/secrets'
