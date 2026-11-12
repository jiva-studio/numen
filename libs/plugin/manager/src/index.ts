/** The plugin list and the plugin page: data in by props, answers out as events. */
export { default as PluginsList } from './PluginsList.vue'
export { default as PluginDetail } from './PluginDetail.vue'
export { default as PluginSettingsCard } from './PluginSettingsCard.vue'
export type { PluginDetailProps } from './PluginDetail.types'
export {
  createManagedPlugins,
  isInstalled,
  OTHER_CATEGORY,
  type ManagedPlugin,
  type PluginListing,
} from './plugins'
export type {
  PluginFailure,
  PluginInfo,
  PluginReader,
  PluginSettingCondition,
  PluginSettingInfo,
  SettingValue,
  PluginSettingOption,
  PluginSettingOptionsBy,
  PluginSettingsViewInfo,
  PluginSettingType,
} from './pluginInfo'
export {
  formatCategoryLabel,
  getFilteredPluginItems,
  getPluginCategory,
  matchesPluginQuery,
  parsePluginQuery,
  type FlatPluginItem,
  type ParsedPluginQuery,
  type PluginFilterContext,
} from './filter'
export { formatFailedAt, getFailedPlugins, getPluginState, type PluginState } from './state'
export { parsePluginReadme, type PluginReadmeParts } from './readme'
export { PLUGIN_WORDS } from './words'
